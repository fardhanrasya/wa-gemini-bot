package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// ==========================================================================
// Donation — logika fitur donasi, dipisah dari bot.go.
//
// File ini berisi semua yang berkaitan dengan alur donasi:
//   - Regex parsing perintah "donasi {jumlah}"
//   - Pembuatan checkout payment via DokuService
//   - Callback saat pembayaran berhasil (kirim terima kasih)
//   - Utility formatRupiah
//
// Pemisahan ini mengikuti prinsip "different layer, different abstraction":
// bot.go menangani routing pesan WhatsApp, donation.go menangani
// business logic donasi. Keduanya bergantung pada DokuService tapi
// dengan concern yang berbeda.
// ==========================================================================

// donationRegex mencocokkan perintah "donasi 10000", "donasi 50000", dll.
// Case-insensitive, hanya angka (tanpa titik/koma) — sesuai penggunaan
// kasual di WhatsApp Indonesia.
var donationRegex = regexp.MustCompile(`(?i)^donasi\s+(\d+)$`)

// handleDonation memeriksa dan memproses perintah donasi.
// Return true jika pesan adalah perintah donasi (sudah di-handle),
// false jika bukan — sehingga bot.go bisa melanjutkan ke handler lain.
func (b *Bot) handleDonation(v *eventContext, cleanText string) bool {
	matches := donationRegex.FindStringSubmatch(cleanText)
	if matches == nil {
		return false
	}

	amount := matches[1]
	log.Printf("[DONASI] %s ingin donasi Rp %s", v.senderName, amount)

	invoiceNumber := fmt.Sprintf("DNR%d", time.Now().UnixMilli())

	// CreatePayment menggabungkan checkout + tracking dalam satu panggilan.
	// Caller tidak perlu ingat urutan — "deeper interface".
	result, err := b.doku.CreatePayment(
		invoiceNumber, amount, v.senderName, v.chatJID, b.config.DokuWebhookURL,
	)
	if err != nil {
		log.Printf("[DONASI] Gagal buat checkout: %v", err)
		b.sendReply(v.msg, "Maaf, gagal membuat link donasi. Coba lagi nanti ya. 😞")
		return true
	}

	reply := fmt.Sprintf(
		"🎁 *DONASI dari %s*\n\n"+
			"Jumlah: *Rp %s*\n\n"+
			"Silakan klik link berikut untuk membayar:\n%s\n\n"+
			"⏰ Link berlaku selama 30 menit.\n"+
			"💳 Bisa bayar pakai QRIS, Transfer Bank, E-Wallet, dll.\n\n"+
			"Terima kasih atas niat baikmu! 🙏",
		v.senderName,
		formatRupiah(amount),
		result.PaymentURL,
	)

	b.sendReply(v.msg, reply)
	b.memory.Record(v.chatJID, "Abdul (Bot)",
		fmt.Sprintf("[Link donasi dibuat untuk %s sebesar Rp %s]", v.senderName, amount))

	return true
}

// handlePaymentSuccess dipanggil oleh DokuService (via callback) saat
// webhook pembayaran masuk. Mengirim pesan terima kasih ke grup WhatsApp.
//
// Fungsi ini berjalan di goroutine webhook server, bukan di goroutine
// event handler WhatsApp — tapi aman karena client.SendMessage thread-safe.
func (b *Bot) handlePaymentSuccess(chatJID, senderName, amount string) {
	jid, err := types.ParseJID(chatJID)
	if err != nil {
		log.Printf("[DONASI] Gagal parse JID %s: %v", chatJID, err)
		return
	}

	reply := fmt.Sprintf(
		"🎉✨ *DONASI BERHASIL!* ✨🎉\n\n"+
			"Terima kasih *%s* atas donasi sebesar *Rp %s* nya! 🙏💖\n\n"+
			"Semoga rezekinya dilancarkan dan berkah selalu. Aamiin! 🤲",
		senderName,
		formatRupiah(amount),
	)

	if _, err := b.client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(reply),
	}); err != nil {
		log.Printf("[DONASI] Gagal kirim pesan terima kasih: %v", err)
	}

	b.memory.Record(chatJID, "Abdul (Bot)",
		fmt.Sprintf("[Donasi dari %s sebesar Rp %s berhasil diterima]", senderName, amount))
}

// formatRupiah memformat angka string menjadi format ribuan Indonesia.
// Contoh: "50000" → "50.000", "1000000" → "1.000.000"
func formatRupiah(amount string) string {
	n := len(amount)
	if n <= 3 {
		return amount
	}

	var result strings.Builder
	remainder := n % 3
	if remainder > 0 {
		result.WriteString(amount[:remainder])
		result.WriteString(".")
	}

	for i := remainder; i < n; i += 3 {
		if i > remainder {
			result.WriteString(".")
		}
		result.WriteString(amount[i : i+3])
	}

	return result.String()
}
