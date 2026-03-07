package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// chatMessage merepresentasikan satu pesan di riwayat grup.
// Unexported karena detail internal — caller hanya perlu Record() dan GetContext().
type chatMessage struct {
	Sender    string
	Text      string
	Timestamp time.Time
}

// GroupMemory mengelola riwayat obrolan per grup menggunakan sliding window.
//
// Desain: interface-nya sengaja dibuat sangat simpel (Record + GetContext) supaya
// caller tidak perlu tahu tentang mutex, sliding window, atau format internal.
// Ini adalah contoh "deep module" — interface kecil, kompleksitas tersembunyi.
type GroupMemory struct {
	maxMessages int
	history     map[string][]chatMessage
	mu          sync.Mutex
}

// NewGroupMemory membuat GroupMemory baru dengan batas maxMessages per grup.
func NewGroupMemory(maxMessages int) *GroupMemory {
	return &GroupMemory{
		maxMessages: maxMessages,
		history:     make(map[string][]chatMessage),
	}
}

// Record menyimpan satu pesan baru ke riwayat sebuah grup.
// Thread-safe — bisa dipanggil dari goroutine manapun tanpa khawatir race condition.
// Otomatis membuang pesan paling tua jika sudah melebihi batas.
func (m *GroupMemory) Record(groupJID, sender, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := chatMessage{
		Sender:    sender,
		Text:      text,
		Timestamp: time.Now(),
	}

	h := m.history[groupJID]
	h = append(h, msg)
	if len(h) > m.maxMessages {
		h = h[len(h)-m.maxMessages:]
	}
	m.history[groupJID] = h
}

// GetContext mengembalikan riwayat obrolan sebagai string yang siap
// digabungkan ke prompt AI. Format output adalah detail internal yang
// disembunyikan dari caller — mereka hanya perlu tahu bahwa hasilnya
// adalah konteks yang bisa langsung dipakai.
func (m *GroupMemory) GetContext(groupJID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	h, ok := m.history[groupJID]
	if !ok || len(h) == 0 {
		return "(Tidak ada riwayat obrolan sebelumnya)"
	}

	var sb strings.Builder
	sb.WriteString("=== RIWAYAT OBROLAN GRUP ===\n")
	for _, msg := range h {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp.Format("15:04"), msg.Sender, msg.Text))
	}
	sb.WriteString("=== AKHIR RIWAYAT ===")
	return sb.String()
}
