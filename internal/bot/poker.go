package bot

import (
	"context"
	"log"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// ==========================================================================
// Poker — logika integrasi poker dengan Bot, dipisah dari bot.go.
//
// File ini berisi:
//   - handlePokerMention: dispatch command poker yang di-mention
//   - sendDM: kirim pesan privat ke player (untuk hole cards)
//
// Aksi poker tanpa mention (fold/call/raise/check/bet/allin) di-intercept
// langsung di bot.go sebelum mention check. Lihat handleMessage().
//
// Pemisahan ini mengikuti pola yang sama dengan donation.go:
// bot.go menangani routing, poker.go menangani bridge ke PokerService.
// ==========================================================================

// handlePokerMention memproses command poker yang di-mention.
// Contoh: "@Abdul poker", "@Abdul ikut", "@Abdul mulai", "@Abdul status"
//
// Return true jika pesan di-handle sebagai perintah poker.
func (b *Bot) handlePokerMention(ctx *eventContext) bool {
	if b.poker == nil {
		return false
	}

	senderJID := ctx.msg.Info.Sender.ToNonAD().String()
	return b.poker.HandleMentionCommand(ctx.chatJID, ctx.senderName, senderJID, ctx.cleanText)
}

// sendDM mengirim pesan privat (Direct Message) ke seorang user.
// Digunakan oleh PokerService via callback untuk mengirim hole cards.
//
// Sender JID dari WhatsApp bisa berisi device part (misal "12345:45@lid")
// yang harus distrip sebelum mengirim DM. WhatsApp mengharuskan
// recipient JID berupa user JID tanpa device part.
func (b *Bot) sendDM(userJID, text string) {
	jid, err := types.ParseJID(userJID)
	if err != nil {
		log.Printf("[POKER] Gagal parse JID untuk DM %s: %v", userJID, err)
		return
	}

	// Strip device part — DM harus ke user JID tanpa device.
	// Tanpa ini, whatsmeow menolak dengan error:
	// "message recipient must be a user JID with no device part"
	jid = types.NewJID(jid.User, jid.Server)

	if _, err := b.client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(text),
	}); err != nil {
		log.Printf("[POKER] Gagal kirim DM ke %s: %v", jid.String(), err)
	}
}

