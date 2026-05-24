package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"wa-gemini-bot/internal/ai"
	"wa-gemini-bot/internal/config"
	"wa-gemini-bot/internal/economy"
	"wa-gemini-bot/internal/media"
	"wa-gemini-bot/internal/memory"
	"wa-gemini-bot/internal/blackjack"
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
	trivia    *trivia.TriviaService // nil jika fitur trivia tidak aktif
	poker     *poker.PokerService   // nil jika fitur poker tidak aktif
	blackjack *blackjack.BlackjackService // nil jika fitur blackjack tidak aktif
	eco       *economy.EconomyService
	cld    *media.CloudinaryService // nil jika tidak dikonfigurasi
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
func NewBot(cfg *config.Config, ai *ai.AIService, mem *memory.GroupMemory, doku *payment.DokuService, trivia *trivia.TriviaService, poker *poker.PokerService, blackjack *blackjack.BlackjackService, eco *economy.EconomyService, cld *media.CloudinaryService) (*Bot, error) {
	ctx := context.Background()

	dbLog := waLog.Stdout("Database", "WARN", true)
	container, err := sqlstore.New(ctx, "sqlite3", "file:data/wa-session.db?_foreign_keys=on", dbLog)
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
		trivia:    trivia,
		poker:     poker,
		blackjack: blackjack,
		eco:       eco,
		cld:       cld,
	}

	client.AddEventHandler(bot.handleEvent)

	// Daftarkan callback pembayaran — mengunci siklus "DOKU → Bot → WhatsApp"
	if doku != nil {
		doku.SetPaymentCallback(bot.handlePaymentSuccess)
	}

	// Daftarkan callback trivia — siklus "Trivia → Bot → WhatsApp"
	if trivia != nil {
		trivia.SetCallbacks(bot.sendToGroup, bot.sendToGroupWithMentions, bot.sendPollToGroup, mem.Record, eco.AddBalance)
	}

	// Daftarkan callback poker — siklus "Poker → Bot → WhatsApp"
	if poker != nil {
		poker.SetCallbacks(bot.sendToGroup, bot.sendToGroupWithMentions, bot.sendDM, mem.Record, eco.AddBalance, eco.SubtractBalance)
	}

	// Daftarkan callback blackjack — siklus "Blackjack → Bot → WhatsApp"
	if blackjack != nil {
		blackjack.SetCallbacks(bot.sendToGroup, bot.sendToGroupWithMentions, bot.sendDM, mem.Record, eco.AddBalance, eco.SubtractBalance)
	}

	return bot, nil
}

// Start menghubungkan bot ke WhatsApp dan menjalankan HTTP server (DOKU Webhook + Admin Panel).
func (b *Bot) Start() error {
	// Jalankan HTTP server gabungan secara background
	go b.StartHTTPServer(b.config.DokuWebhookPort)

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

	rawAction := strings.TrimSpace(rawText)
	senderJID := v.Info.Sender.ToNonAD().String()

	// Blackjack bet/leave tanpa mention — sebelum poker agar "bet N" tidak masuk aksi poker.
	if b.blackjack != nil && b.blackjack.HandleQuickCommand(chatJID, senderName, senderJID, rawAction) {
		return
	}

	// Poker game actions (fold/call/raise/check/bet/allin) — tanpa mention.
	if b.poker != nil && b.poker.IsActive(chatJID) {
		if b.poker.HandleGameAction(chatJID, senderName, rawAction) {
			return
		}
	}

	// Blackjack hit/stand/double — tanpa mention, saat giliran aktif.
	if b.blackjack != nil && b.blackjack.IsActive(chatJID) {
		if b.blackjack.HandleGameAction(chatJID, senderName, rawAction) {
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

	// Interseptor perintah leave/keluar kontekstual (diawali @mention bot)
	cleanLower := strings.ToLower(cleanText)
	if cleanLower == "leave" || cleanLower == "keluar" {
		senderJID := ctx.msg.Info.Sender.ToNonAD().String()
		if b.blackjack != nil && b.blackjack.IsPlayerInGame(chatJID, senderJID) {
			ctx.cleanText = "bj leave"
		} else if b.poker != nil && b.poker.IsPlayerInGame(chatJID, senderName) {
			// Biarkan "leave"/"keluar" untuk di-handle poker
		} else {
			// Fallback jika tidak aktif di game mana pun tetapi salah satu game aktif di grup
			if b.blackjack != nil && b.blackjack.IsActive(chatJID) {
				ctx.cleanText = "bj leave"
			}
		}
	}

	// Dispatch ke handler yang sesuai — poker & blackjack mention commands diperiksa dulu
	if b.handlePokerMention(ctx) {
		return
	}

	if b.handleBlackjackMention(ctx) {
		return
	}

	if b.doku != nil && b.handleDonation(ctx, cleanText) {
		return
	}

	if b.cld != nil && strings.HasPrefix(strings.ToLower(cleanText), "upscale") {
		b.handleUpscale(ctx)
		return
	}

	if b.handleEconomy(ctx) {
		return
	}

	b.handleAIQuery(ctx)
}

// handleEconomy menangani command terkait saldo (saldo, transfer).
func (b *Bot) handleEconomy(ctx *eventContext) bool {
	if b.eco == nil {
		return false
	}

	cleanText := ctx.cleanText

	// Selalu coba update nama jika berinteraksi dengan economy
	senderJID := ctx.msg.Info.Sender.ToNonAD().String()
	_ = b.eco.UpdateName(senderJID, ctx.senderName)

	if cleanText == "help" || cleanText == "menu" || cleanText == "bantuan" || cleanText == "tolong" {
		var sb strings.Builder
		sb.WriteString("🤖 *ABDUL BOT — DAFTAR PERINTAH* 🤖\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("Halo! Saya adalah bot asisten grup Anda. Berikut adalah daftar semua perintah yang bisa Anda gunakan:\n\n")

		sb.WriteString("💰 *EKONOMI & PROFIL*\n")
		sb.WriteString("* `@bot saldo` : Cek saldo chip dan pangkat aktif Anda.\n")
		sb.WriteString("* `@bot leaderboard` : Tampilkan 10 pemain terkaya di grup ini.\n")
		sb.WriteString("* `@bot ranks` : Tampilkan seluruh daftar pangkat dan persyaratannya.\n")
		sb.WriteString("* `@bot setname <nama>` : Ubah nama tampilan Anda di bot (maksimal 20 karakter).\n")
		sb.WriteString("* `@bot setname reset` : Kembalikan nama tampilan mengikuti profil WhatsApp Anda.\n")
		sb.WriteString("* `@bot transfer @user <jumlah>` : Transfer chip Anda ke anggota grup lain.\n\n")

		sb.WriteString("🃏 *POKER (TEXAS HOLD'EM)*\n")
		sb.WriteString("* `@bot poker` : Membuat lobby game.\n")
		sb.WriteString("* `@bot poker guide` : Tampilkan bantuan lengkap & panduan bermain poker.\n")
		sb.WriteString("* `@bot poker help` : Tampilkan semua command dalam poker.\n")
		sb.WriteString("* `@bot ikut` : Join ke dalam antrean/lobby permainan poker di grup.\n")
		sb.WriteString("* `@bot mulai` : Memulai permainan poker jika lobby memiliki minimal 2 pemain.\n")
		sb.WriteString("* `@bot leave` : Keluar dari meja poker.\n")
		sb.WriteString("* `@bot status` : Tampilkan status game/lobby poker aktif saat ini.\n\n")

		sb.WriteString("🎰 *BLACKJACK (21)*\n")
		sb.WriteString("* `@bot bj` : Membuat lobby game blackjack baru.\n")
		sb.WriteString("* `@bot bj guide` : Tampilkan panduan cara bermain blackjack.\n")
		sb.WriteString("* `@bot bj help` : Tampilkan semua command dalam blackjack.\n")
		sb.WriteString("* `@bot bj ikut <saldo>` : Beli/top-up saldo meja blackjack (seperti poker).\n")
		sb.WriteString("* `@bot bj bet <jumlah>` : Atur taruhan (sama seperti `bet <jumlah>` tanpa tag).\n")
		sb.WriteString("* `@bot bj mulai` : Memulai permainan blackjack secara manual.\n")
		sb.WriteString("* `@bot bj status` : Tampilkan status game blackjack aktif.\n")
		sb.WriteString("* `@bot bj leave` : Keluar meja (sama seperti `leave` tanpa tag).\n")
		sb.WriteString("* Tanpa tag (saat di meja BJ): `hit` `stand` `double` | jeda ronde: `bet <jumlah>` `leave`\n\n")

		sb.WriteString("🖼️ *UTILITY*\n")
		sb.WriteString("* `@bot upscale` : Tingkatkan resolusi/kualitas gambar (Gunakan sebagai balasan/reply pada gambar).\n\n")

		sb.WriteString("🎁 *DONASI*\n")
		sb.WriteString("* `@bot donasi <jumlah>` : Berdonasi via DOKU Payment (Mendukung QRIS, e-Wallet, dll).\n\n")

		sb.WriteString("💬 *KECERDASAN BUATAN (AI)*\n")
		sb.WriteString("* `@bot <pertanyaan>` : Tanya apa saja kepada AI! Cukup tag/mention saya di awal pesan.\n\n")

		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("💡 *Tips*: Kuis trivia otomatis akan muncul di grup ini secara berkala. Jawab dengan memilih opsi pada Polling untuk memenangkan chip gratis!")

		b.sendToGroup(ctx.chatJID, sb.String())
		return true
	}

	if strings.HasPrefix(cleanText, "setname ") || strings.HasPrefix(cleanText, "nickname ") {
		var nameArg string
		if strings.HasPrefix(cleanText, "setname ") {
			nameArg = strings.TrimPrefix(cleanText, "setname ")
		} else {
			nameArg = strings.TrimPrefix(cleanText, "nickname ")
		}
		newName := strings.TrimSpace(nameArg)
		if newName == "" {
			b.sendToGroup(ctx.chatJID, "❌ Format salah. Gunakan: `@bot setname <nama_baru>` atau `@bot setname reset`")
			return true
		}

		if strings.ToLower(newName) == "reset" {
			if err := b.eco.ResetCustomName(senderJID); err != nil {
				b.sendToGroup(ctx.chatJID, fmt.Sprintf("❌ Gagal mereset nama kustom: %v", err))
			} else {
				b.sendToGroup(ctx.chatJID, fmt.Sprintf("✅ Nama kustom di-reset. Nama kamu sekarang otomatis mengikuti profil WA: *%s*", ctx.senderName))
			}
			return true
		}

		// Batasi panjang nama agar tidak merusak tampilan chat/leaderboard (maksimal 20 karakter)
		if len([]rune(newName)) > 20 {
			b.sendToGroup(ctx.chatJID, "❌ Nama kustom terlalu panjang. Maksimal 20 karakter.")
			return true
		}

		if err := b.eco.SetCustomName(senderJID, newName); err != nil {
			b.sendToGroup(ctx.chatJID, fmt.Sprintf("❌ Gagal mengubah nama kustom: %v", err))
		} else {
			b.sendToGroup(ctx.chatJID, fmt.Sprintf("✅ Nama kustom kamu berhasil diubah menjadi: *%s*", newName))
		}
		return true
	}

	if cleanText == "saldo" || cleanText == "balance" {
		user, err := b.eco.GetUser(senderJID)
		if err != nil {
			b.sendToGroup(ctx.chatJID, fmt.Sprintf("❌ Gagal mengecek saldo: %v", err))
			return true
		}
		displayName := user.Name
		if displayName == "" {
			displayName = ctx.senderName
		}
		rank := economy.GetRankByBalance(user.Balance)
		b.sendToGroup(ctx.chatJID, fmt.Sprintf("💰 Saldo *%s* saat ini: *%s* chip [%s]", displayName, formatNumber(user.Balance), rank.Styled))
		return true
	}

	if cleanText == "ranks" || cleanText == "pangkat" || cleanText == "leaderboard help" || cleanText == "leaderboard info" {
		var sb strings.Builder
		sb.WriteString("🎖️ *DAFTAR PANGKAT* 🎖️\n")
		sb.WriteString("Kumpulkan chip untuk naik pangkat!\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
		for _, r := range economy.Ranks {
			var limitStr string
			if r.MinChips == 0 {
				limitStr = "0 - 4.999 chip"
			} else {
				limitStr = fmt.Sprintf(">= %s chip", formatNumber(r.MinChips))
			}
			sb.WriteString(fmt.Sprintf("%s (%s)\n_%s_\n\n", r.Styled, limitStr, r.Description))
		}
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("💡 *Tips*: Ketik `@bot setname <nama>` untuk mengganti nickname kustom profil bot kamu!")
		b.sendToGroup(ctx.chatJID, sb.String())
		return true
	}

	if cleanText == "leaderboard" || cleanText == "top" {
		users, err := b.eco.GetLeaderboard(10)
		if err != nil {
			b.sendToGroup(ctx.chatJID, fmt.Sprintf("❌ Gagal mengambil leaderboard: %v", err))
			return true
		}

		var sb strings.Builder
		sb.WriteString("🏆 *LEADERBOARD* 🏆\n━━━━━━━━━━━━━━━━━━━━━━\n")
		for i, u := range users {
			name := u.Name
			if name == "" {
				name = "Unknown"
			}
			rank := economy.GetRankByBalance(u.Balance)
			sb.WriteString(fmt.Sprintf("%d. %s *%s*: 💰 %s chip\n", i+1, rank.Styled, name, formatNumber(u.Balance)))
		}
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━")
		b.sendToGroup(ctx.chatJID, sb.String())
		return true
	}

	if strings.HasPrefix(cleanText, "transfer ") {
		parts := strings.Split(cleanText, " ")
		if len(parts) < 3 {
			b.sendToGroup(ctx.chatJID, "❌ Format salah. Gunakan: @Abdul transfer @user jumlah")
			return true
		}

		// Karena teks sudah di-clean (mention menjadi "@<nomor>"),
		// target ada di parts[1] (misal "@628...") dan jumlah di parts[2].
		// Namun yang lebih akurat adalah mengambil MentionedJid dari pesan asli.
		var targetJID string
		if extMsg := ctx.msg.Message.GetExtendedTextMessage(); extMsg != nil && extMsg.GetContextInfo() != nil {
			mentions := extMsg.GetContextInfo().GetMentionedJID()
			// Hapus bot sendiri dari daftar mention (karena memicu command ini)
			myIDs := b.myIdentifiers()
			for _, m := range mentions {
				isMe := false
				for _, myID := range myIDs {
					if strings.Contains(m, myID) {
						isMe = true
						break
					}
				}
				if !isMe {
					targetJID = m
					break
				}
			}
		}

		if targetJID == "" {
			b.sendToGroup(ctx.chatJID, "❌ Kamu harus mention pemain yang ingin ditransfer. (@user)")
			return true
		}

		amountStr := parts[len(parts)-1]
		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount <= 0 {
			b.sendToGroup(ctx.chatJID, "❌ Jumlah transfer tidak valid.")
			return true
		}

		senderJID := ctx.msg.Info.Sender.ToNonAD().String()
		if err := b.eco.Transfer(senderJID, targetJID, amount); err != nil {
			b.sendToGroup(ctx.chatJID, fmt.Sprintf("❌ Transfer gagal: %v", err))
			return true
		}

		b.sendToGroup(ctx.chatJID, fmt.Sprintf("✅ Berhasil transfer %s chip ke target!", formatNumber(amount)))
		return true
	}

	return false
}

// formatNumber memformat angka dengan pemisah ribuan.
func formatNumber(n int) string {
	in := strconv.Itoa(n)
	var out []rune
	for i, r := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, r)
	}
	return string(out)
}

// handleUpscale menangani request untuk upscale gambar menggunakan Cloudinary.
func (b *Bot) handleUpscale(ctx *eventContext) {
	if ctx.imageData == nil {
		b.sendReply(ctx.msg, "❌ Kamu harus mengirim gambar atau me-reply gambar dengan caption '@Abdul upscale' untuk menggunakan fitur ini.")
		return
	}

	b.sendReply(ctx.msg, "⏳ Sedang mengupscale gambar, tunggu sebentar ya...")

	upscaledData, err := b.cld.UpscaleImage(context.Background(), ctx.imageData)
	if err != nil {
		log.Printf("Error upscale: %v", err)
		b.sendReply(ctx.msg, "❌ Gagal mengupscale gambar. Mungkin ukurannya terlalu besar atau sedang ada gangguan.")
		return
	}

	if err := b.sendImage(ctx.msg, upscaledData, "image/jpeg", "✅ Ini hasil upscalenya!"); err != nil {
		log.Printf("Gagal mengirim gambar upscale: %v", err)
		b.sendReply(ctx.msg, "❌ Gagal mengirim gambar balasan.")
	}
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

// sendImage mengirim gambar balasan ke chat yang sama.
func (b *Bot) sendImage(v *events.Message, imageData []byte, mimeType, caption string) error {
	resp, err := b.client.Upload(context.Background(), imageData, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("gagal upload media ke whatsapp: %w", err)
	}

	msg := &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
		},
	}

	_, err = b.client.SendMessage(context.Background(), v.Info.Chat, msg)
	return err
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
	senderJID := v.Info.Sender.ToNonAD().String()

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
