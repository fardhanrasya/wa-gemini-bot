package bot

// handleBlackjackMention memproses command blackjack yang di-mention.
// Contoh: "@Abdul bj", "@Abdul bj ikut 500", "@Abdul bj mulai", "@Abdul bj status"
//
// Return true jika pesan di-handle sebagai perintah blackjack.
func (b *Bot) handleBlackjackMention(ctx *eventContext) bool {
	if b.blackjack == nil {
		return false
	}

	senderJID := ctx.msg.Info.Sender.ToNonAD().String()
	return b.blackjack.HandleMentionCommand(ctx.chatJID, ctx.senderName, senderJID, ctx.cleanText)
}
