package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"wa-gemini-bot/internal/ai"
	"wa-gemini-bot/internal/config"
	"wa-gemini-bot/internal/memory"
	"wa-gemini-bot/internal/payment"
	"wa-gemini-bot/internal/poker"
	"wa-gemini-bot/internal/trivia"

	"github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Trigger sqlite3 driver registration
var _ sqlite3.SQLiteDriver

// Bot mengelola WhatsApp client dan menghubungkannya dengan AI, memori, dan payment.
//
// Desain: Bot adalah "orchestrator" yang menghubungkan modul-modul independen.
// Setiap modul (AI, Memory, Doku) tidak saling kenal — mereka hanya berkomunikasi
// melalui Bot. Ini menerapkan prinsip "separate general-purpose from special-purpose":
// modul-modul bersifat general-purpose, Bot yang menambahkan konteks WhatsApp.
type Bot struct {
	client *whatsmeow.Client
	ai     *ai.AIService
	memory *memory.GroupMemory
	config *config.Config
	doku   *payment.DokuService   // nil jika fitur donasi tidak aktif
	trivia *trivia.TriviaService // nil jika fitur trivia tidak aktif
	poker  *poker.PokerService   // nil jika fitur poker tidak aktif
}

// eventContext mengumpulkan data yang sudah di-parse dari satu pesan masuk.
// Menghindari "pass-through" parameter yang sama ke banyak method —
// sesuai prinsip mengurangi complexity yang terlihat di interface.
type eventContext struct {
	msg        *events.Message
	chatJID    string
	senderName string
	cleanText  string
	imageData  []byte
	mimeType   string
}

// NewBot membuat Bot baru yang siap di-Start().
// Constructor ini "deep" — menyembunyikan semua setup database, device store,
// dan event wiring. Caller hanya perlu pass dependency, lalu Start().
func NewBot(cfg *config.Config, ai *ai.AIService, mem *memory.GroupMemory, doku *payment.DokuService, trivia *trivia.TriviaService, poker *poker.PokerService) (*Bot, error) {
	ctx := context.Background()

	dbLog := waLog.Stdout("Database", "WARN", true)
	container, err := sqlstore.New(ctx, "sqlite3", "file:wa-session.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("gagal setup database sesi: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil device store: %w", err)
	}

	clientLog := waLog.Stdout("Client", "WARN", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	bot := &Bot{
		client: client,
		ai:     ai,
		memory: mem,
		config: cfg,
		doku:   doku,
		trivia: trivia,
		poker:  poker,
	}

	client.AddEventHandler(bot.handleEvent)

	// Daftarkan callback pembayaran — mengunci siklus "DOKU → Bot → WhatsApp"
	if doku != nil {
		doku.SetPaymentCallback(bot.handlePaymentSuccess)
	}

	// Daftarkan callback trivia — siklus "Trivia → Bot → WhatsApp"
	if trivia != nil {
		trivia.SetCallbacks(bot.sendToGroup, bot.sendToGroupWithMentions, bot.sendPollToGroup, mem.Record)
	}

	// Daftarkan callback poker — siklus "Poker → Bot → WhatsApp"
	if poker != nil {
		poker.SetCallbacks(bot.sendToGroup, bot.sendToGroupWithMentions, bot.sendDM, mem.Record)
	}

	return bot, nil
}

// Start menghubungkan bot ke WhatsApp.
// Jika belum login, menampilkan QR code untuk di-scan.
func (b *Bot) Start() error {
	if b.client.Store.ID == nil {
		return b.connectWithQR()
	}
	return b.client.Connect()
}

// Stop memutus koneksi WhatsApp secara graceful.
func (b *Bot) Stop() {
	b.client.Disconnect()
}

// connectWithQR menampilkan QR code dan menunggu login berhasil.
func (b *Bot) connectWithQR() error {
	qrChan, _ := b.client.GetQRChannel(context.Background())
	if err := b.client.Connect(); err != nil {
		return fmt.Errorf("gagal connect untuk QR: %w", err)
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			fmt.Println("SCAN QR CODE INI DI WHATSAPP KAMU:")
			fmt.Println(evt.Code)
		} else {
			log.Printf("Login event: %s", evt.Event)
		}
	}
	return nil
}

// ==========================================================================
// Message handling — alur pemrosesan pesan menggunakan "guard clauses"
// yang mengeliminasi kasus-kasus tidak relevan lebih dulu, sehingga
// happy path tetap flat dan mudah dibaca (tidak nested).
// ==========================================================================

func (b *Bot) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		b.handleMessage(v)
	}
}

func (b *Bot) handleMessage(v *events.Message) {
	if v.Info.IsFromMe || !v.Info.IsGroup {
		return
	}

	chatJID := v.Info.Chat.String()
	if !b.config.IsAllowedGroup(chatJID) {
		return
	}

	// Cek jika ini adalah update polling (vote)
	if v.Message.GetPollUpdateMessage() != nil {
		b.handlePollUpdate(v)
		return
	}

	senderName := senderNameFrom(v)
	rawText := rawTextFrom(v)

	// Rekam semua pesan — bot "menyimak" meskipun tidak di-tag
	if rawText != "" {
		b.memory.Record(chatJID, senderName, rawText)
		log.Printf("[REKAM] %s: %s", senderName, rawText)
	}

	// Poker game actions (fold/call/raise/check/bet/allin) — intercept SEBELUM
	// mention check karena aksi ini tidak perlu mention bot.
	// Hanya dicek jika ada game poker aktif di grup ini.
	if b.poker != nil && b.poker.IsActive(chatJID) {
		rawAction := strings.TrimSpace(rawText)
		if b.poker.HandleGameAction(chatJID, senderName, rawAction) {
			return
		}
	}

	if !b.isMentioningMe(v) {
		return
	}

	cleanText := b.cleanedTextFrom(v)
	// Jika tidak ada teks dan bukan pesan gambar, abaikan.
	// Tapi jika ada gambar, kita tetap proses (mungkin user cuma tag "ini apa?").
	if cleanText == "" && v.Message.GetImageMessage() == nil && v.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage().GetImageMessage() == nil {
		return
	}

	log.Printf("[TANYA] %s: %s", senderName, cleanText)

	// Deteksi gambar (baik langsung atau via reply)
	var imageData []byte
	var mimeType string
	if img := v.Message.GetImageMessage(); img != nil {
		data, err := b.client.Download(context.Background(), img)
		if err != nil {
			log.Printf("Gagal download gambar: %v", err)
		} else {
			imageData = data
			mimeType = img.GetMimetype()
		}
	} else if quoted := v.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage(); quoted != nil {
		if img := quoted.GetImageMessage(); img != nil {
			data, err := b.client.Download(context.Background(), img)
			if err != nil {
				log.Printf("Gagal download gambar quoted: %v", err)
			} else {
				imageData = data
				mimeType = img.GetMimetype()
			}
		}
	}

	// Bangun event context — menghindari pass-through banyak parameter
	ctx := &eventContext{
		msg:        v,
		chatJID:    chatJID,
		senderName: senderName,
		cleanText:  cleanText,
		imageData:  imageData,
		mimeType:   mimeType,
	}

	// Dispatch ke handler yang sesuai — poker mention commands diperiksa dulu
	if b.handlePokerMention(ctx) {
		return
	}

	if b.doku != nil && b.handleDonation(ctx, cleanText) {
		return
	}

	b.handleAIQuery(ctx)
}

// handleAIQuery mengirim pertanyaan ke AI dengan konteks obrolan grup.
func (b *Bot) handleAIQuery(ctx *eventContext) {
	cleanText := ctx.cleanText
	if cleanText == "" && ctx.imageData != nil {
		cleanText = "Deskripsikan gambar ini secara detail atau jawab sesuai konteks jika ini adalah bagian dari percakapan."
	}

	prompt := fmt.Sprintf(
		"%s\n\nPesan dari %s:\n%s",
		b.memory.GetContext(ctx.chatJID),
		ctx.senderName,
		cleanText,
	)

	var reply string
	var err error

	if ctx.imageData != nil {
		log.Printf("[VISION] Menganalisis gambar dari %s", ctx.senderName)
		reply, err = b.ai.AnalyzeImage(prompt, ctx.imageData, ctx.mimeType)
	} else {
		reply, err = b.ai.Ask(prompt)
	}

	if err != nil {
		log.Printf("Error AI: %v", err)
		reply = "Waduh, otak AI-ku lagi error nih. 😵"
	}

	b.sendReply(ctx.msg, reply)
	b.memory.Record(ctx.chatJID, "Abdul (Bot)", reply)
}

// ==========================================================================
// WhatsApp utilities — fungsi-fungsi kecil yang mengekstrak data dari
// pesan WhatsApp. Dipisah agar handleMessage tetap readable.
// ==========================================================================

// sendReply mengirim pesan balasan ke chat yang sama.
func (b *Bot) sendReply(v *events.Message, text string) {
	if _, err := b.client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		Conversation: proto.String(text),
	}); err != nil {
		log.Printf("Gagal kirim pesan: %v", err)
	}
}

// sendToGroup mengirim pesan ke grup berdasarkan JID string.
// Dipakai oleh TriviaService via callback.
func (b *Bot) sendToGroup(groupJID, text string) {
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		log.Printf("Gagal parse JID %s: %v", groupJID, err)
		return
	}
	if _, err := b.client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(text),
	}); err != nil {
		log.Printf("Gagal kirim pesan ke %s: %v", groupJID, err)
	}
}

// sendToGroupWithMentions mengirim pesan ke grup dengan proper @-mention.
//
// WhatsApp mentions butuh dua hal:
//  1. Teks yang mengandung "@<nomor>" (tampilan visual)
//  2. MentionedJid di ContextInfo (agar notifikasi muncul)
//
// mentionJIDs berisi JID lengkap (mungkin ada device part) — kita strip
// device part-nya sebelum kirim, sama seperti di sendDM.
func (b *Bot) sendToGroupWithMentions(groupJID, text string, mentionJIDs []string) {
	if len(mentionJIDs) == 0 {
		// Fallback ke plain text jika tidak ada mention
		b.sendToGroup(groupJID, text)
		return
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		log.Printf("Gagal parse JID %s: %v", groupJID, err)
		return
	}

	// Strip device part dari setiap JID
	cleanJIDs := make([]string, len(mentionJIDs))
	for i, rawJID := range mentionJIDs {
		parsed, err := types.ParseJID(rawJID)
		if err != nil {
			cleanJIDs[i] = rawJID // fallback ke raw
		} else {
			cleanJIDs[i] = types.NewJID(parsed.User, parsed.Server).String()
		}
	}

	if _, err := b.client.SendMessage(context.Background(), jid, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				MentionedJID: cleanJIDs,
			},
		},
	}); err != nil {
		log.Printf("Gagal kirim pesan mention ke %s: %v", groupJID, err)
	}
}

// sendPollToGroup mengirim polling asli ke grup.
func (b *Bot) sendPollToGroup(groupJID, question string, options []string) (string, error) {
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return "", err
	}

	// BuildPollCreation membuat message polling (selectable count 1 = satu jawaban)
	pollMsg := b.client.BuildPollCreation(question, options, 1)

	resp, err := b.client.SendMessage(context.Background(), jid, pollMsg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// handlePollUpdate menangani update polling (saat member memilih).
func (b *Bot) handlePollUpdate(v *events.Message) {
	if b.trivia == nil {
		return
	}

	// Dekripsi isi voting polling
	vote, err := b.client.DecryptPollVote(context.Background(), v)
	if err != nil {
		log.Printf("Gagal dekripsi vote polling: %v", err)
		return
	}

	chatJID := v.Info.Chat.String()
	senderName := senderNameFrom(v)
	senderJID := v.Info.Sender.String()

	// Ambil ID pesan polling asli yang di-update
	pollMsgID := v.Message.GetPollUpdateMessage().GetPollCreationMessageKey().GetID()

	// Kirim data voting ke TriviaService untuk dicocokkan
	b.trivia.RecordAnswer(chatJID, pollMsgID, senderName, senderJID, vote.GetSelectedOptions())
}

// isMentioningMe memeriksa apakah pesan menyebut bot.
// Menggunakan Phone JID dan LID karena WhatsApp menggunakan
// format berbeda tergantung versi dan addressing mode.
func (b *Bot) isMentioningMe(v *events.Message) bool {
	identifiers := b.myIdentifiers()

	var contextInfo *waProto.ContextInfo
	if extMsg := v.Message.GetExtendedTextMessage(); extMsg != nil {
		contextInfo = extMsg.GetContextInfo()
	} else if imgMsg := v.Message.GetImageMessage(); imgMsg != nil {
		contextInfo = imgMsg.GetContextInfo()
	}

	if contextInfo == nil {
		return false
	}

	for _, mentioned := range contextInfo.GetMentionedJID() {
		for _, myID := range identifiers {
			if strings.Contains(mentioned, myID) {
				return true
			}
		}
	}
	return false
}

// myIdentifiers mengembalikan semua identifier yang bisa dipakai
// untuk mencocokkan mention ke bot ini.
func (b *Bot) myIdentifiers() []string {
	var ids []string
	if b.client.Store.ID != nil {
		ids = append(ids, b.client.Store.ID.User)
	}
	if lid := b.client.Store.LID; lid.User != "" {
		ids = append(ids, lid.User)
	}
	return ids
}

// cleanedTextFrom mengambil teks pesan dan menghapus mention tag bot.
func (b *Bot) cleanedTextFrom(v *events.Message) string {
	text := rawTextFrom(v)
	for _, id := range b.myIdentifiers() {
		text = strings.ReplaceAll(text, "@"+id, "")
	}
	return strings.TrimSpace(text)
}

// rawTextFrom mengambil teks mentah dari pesan tanpa modifikasi.
func rawTextFrom(v *events.Message) string {
	if text := v.Message.GetConversation(); text != "" {
		return text
	}
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	if img := v.Message.GetImageMessage(); img != nil {
		return img.GetCaption()
	}
	return ""
}

// senderNameFrom mendapatkan nama pengirim yang human-readable.
// Prioritas: push name (nama kontak) → fallback ke user ID.
func senderNameFrom(v *events.Message) string {
	if name := v.Info.PushName; name != "" {
		return name
	}
	return v.Info.Sender.User
}
