package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Trigger sqlite3 driver registration
var _ sqlite3.SQLiteDriver

// Bot mengelola WhatsApp client dan menghubungkannya dengan AI dan memori.
//
// Desain: Bot bertanggung jawab atas semua interaksi WhatsApp.
// AI dan memori disuntikkan (dependency injection) melalui constructor,
// bukan diakses via global — ini membuat setiap komponen bisa diuji dan
// diganti secara independen.
type Bot struct {
	client *whatsmeow.Client
	ai     *AIService
	memory *GroupMemory
	config *Config
}

// NewBot membuat Bot baru yang siap di-Start().
// Menanggung semua kompleksitas: setup database, device store, dan WhatsApp client.
func NewBot(cfg *Config, ai *AIService, mem *GroupMemory) (*Bot, error) {
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
	}

	client.AddEventHandler(bot.handleEvent)

	return bot, nil
}

// Start menghubungkan bot ke WhatsApp.
// Jika belum login, menampilkan QR code untuk di-scan.
// Jika sudah login sebelumnya, langsung connect.
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

// connectWithQR menampilkan QR code dan menunggu sampai login berhasil.
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
// Event handling — semua logika "apa yang terjadi saat ada pesan" ada di sini.
// Method-method di bawah ini private karena merupakan detail internal Bot.
// ==========================================================================

// handleEvent adalah entry point untuk semua event WhatsApp.
func (b *Bot) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		b.handleMessage(v)
	}
}

// handleMessage memproses satu pesan masuk.
// Alur logikanya disusun sebagai "guard clauses" yang mengeliminasi
// kasus-kasus yang tidak perlu diproses, sehingga happy path tetap flat.
func (b *Bot) handleMessage(v *events.Message) {
	if v.Info.IsFromMe || !v.Info.IsGroup {
		return
	}

	chatJID := v.Info.Chat.String()
	if !b.config.IsAllowedGroup(chatJID) {
		return
	}

	senderName := senderNameFrom(v)
	rawText := rawTextFrom(v)

	// Rekam semua pesan — bot "menyimak" bahkan kalau tidak di-tag
	if rawText != "" {
		b.memory.Record(chatJID, senderName, rawText)
		log.Printf("[REKAM] %s: %s", senderName, rawText)
	}

	// Hanya proses lebih lanjut kalau bot di-mention
	if !b.isMentioningMe(v) {
		return
	}

	cleanText := b.cleanedTextFrom(v)
	if cleanText == "" {
		return
	}

	log.Printf("[TANYA] %s: %s", senderName, cleanText)

	// Bangun prompt dengan konteks obrolan
	prompt := fmt.Sprintf(
		"%s\n\nPesan dari %s:\n%s",
		b.memory.GetContext(chatJID),
		senderName,
		cleanText,
	)

	reply, err := b.ai.Ask(prompt)
	if err != nil {
		log.Printf("Error AI: %v", err)
		reply = "Waduh, otak AI-ku lagi error nih. 😵"
	}

	b.sendReply(v, reply)
	b.memory.Record(chatJID, "Abdul (Bot)", reply)
}

// sendReply mengirim pesan balasan ke chat yang sama.
func (b *Bot) sendReply(v *events.Message, text string) {
	_, err := b.client.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		log.Printf("Gagal kirim pesan: %v", err)
	}
}

// ==========================================================================
// Utility functions — operasi kecil yang mengekstrak data dari pesan WhatsApp.
// Dibuat sebagai fungsi agar mudah dipahami dan diuji secara terpisah.
// ==========================================================================

// isMentioningMe memeriksa apakah pesan menyebut bot.
// Menggunakan baik Phone JID maupun LID karena WhatsApp menggunakan
// format berbeda tergantung versi dan addressing mode.
func (b *Bot) isMentioningMe(v *events.Message) bool {
	identifiers := b.myIdentifiers()

	extMsg := v.Message.GetExtendedTextMessage()
	if extMsg == nil || extMsg.GetContextInfo() == nil {
		return false
	}

	for _, mentioned := range extMsg.GetContextInfo().GetMentionedJID() {
		for _, myID := range identifiers {
			if strings.Contains(mentioned, myID) {
				return true
			}
		}
	}
	return false
}

// myIdentifiers mengembalikan semua identifier yang bisa dipakai untuk
// mencocokkan mention ke bot ini (Phone number + LID).
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
	return v.Message.GetExtendedTextMessage().GetText()
}

// senderNameFrom mendapatkan nama pengirim yang human-readable.
// Prioritas: push name (nama kontak) → fallback ke user ID.
func senderNameFrom(v *events.Message) string {
	if name := v.Info.PushName; name != "" {
		return name
	}
	return v.Info.Sender.User
}
