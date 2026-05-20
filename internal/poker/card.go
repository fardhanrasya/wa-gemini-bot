package poker

import (
	"fmt"
	"math/rand"
	"strings"
)

// ==========================================================================
// Card representation and deck management.
//
// Desain: Suit dan Rank sebagai integer kecil (bukan string) agar comparison
// dan sorting cepat dan type-safe. Method String() menyediakan representasi
// human-readable untuk pesan WhatsApp.
// ==========================================================================

// Suit merepresentasikan jenis kartu (Spade, Heart, Diamond, Club).
type Suit int

const (
	Spade   Suit = iota // ♠
	Heart                // ♥
	Diamond              // ♦
	Club                 // ♣
)

var suitSymbols = [4]string{"♠", "♥", "♦", "♣"}
var suitNames = [4]string{"Spade", "Heart", "Diamond", "Club"}

func (s Suit) String() string { return suitSymbols[s] }
func (s Suit) Name() string   { return suitNames[s] }

// Rank merepresentasikan nilai kartu (2–14, dimana 14=Ace).
// Ace disimpan sebagai 14 agar perbandingan numerik langsung benar
// untuk sebagian besar kasus. Ace-low straight (A-2-3-4-5) ditangani
// secara khusus di hand evaluator.
type Rank int

const (
	Two   Rank = 2
	Three Rank = 3
	Four  Rank = 4
	Five  Rank = 5
	Six   Rank = 6
	Seven Rank = 7
	Eight Rank = 8
	Nine  Rank = 9
	Ten   Rank = 10
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
	Ace   Rank = 14
)

var rankLabels = map[Rank]string{
	Two: "2", Three: "3", Four: "4", Five: "5", Six: "6",
	Seven: "7", Eight: "8", Nine: "9", Ten: "10",
	Jack: "J", Queen: "Q", King: "K", Ace: "A",
}

var rankNames = map[Rank]string{
	Two: "Two", Three: "Three", Four: "Four", Five: "Five",
	Six: "Six", Seven: "Seven", Eight: "Eight", Nine: "Nine",
	Ten: "Ten", Jack: "Jack", Queen: "Queen", King: "King", Ace: "Ace",
}

func (r Rank) String() string { return rankLabels[r] }
func (r Rank) Name() string   { return rankNames[r] }

// Card merepresentasikan satu kartu bermain.
type Card struct {
	Rank Rank
	Suit Suit
}

// String mengembalikan representasi singkat seperti "A♠", "K♥", "10♦".
func (c Card) String() string {
	return c.Rank.String() + c.Suit.String()
}

// FullName mengembalikan nama lengkap seperti "Ace of Spades".
func (c Card) FullName() string {
	return c.Rank.Name() + " of " + c.Suit.Name()
}

// ==========================================================================
// Deck — kumpulan 52 kartu yang bisa di-shuffle dan di-draw.
// ==========================================================================

// Deck merepresentasikan satu set kartu bermain 52 kartu.
// Field position melacak kartu berikutnya yang akan di-draw,
// menghindari kebutuhan untuk slice ulang setiap kali draw.
type Deck struct {
	cards    [52]Card
	position int
}

// NewDeck membuat deck baru yang sudah di-shuffle dan siap di-draw.
func NewDeck() *Deck {
	d := &Deck{}
	i := 0
	for suit := Spade; suit <= Club; suit++ {
		for rank := Two; rank <= Ace; rank++ {
			d.cards[i] = Card{Rank: rank, Suit: suit}
			i++
		}
	}
	d.Shuffle()
	return d
}

// Shuffle mengacak urutan kartu dan reset posisi draw ke awal.
func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
	d.position = 0
}

// Draw mengambil satu kartu dari atas deck.
// Panic jika deck habis — ini seharusnya tidak terjadi dalam permainan
// poker normal (max 8 pemain × 2 + 5 community + 3 burn = 24 kartu).
func (d *Deck) Draw() Card {
	if d.position >= len(d.cards) {
		panic("deck habis — seharusnya tidak terjadi dalam permainan poker normal")
	}
	c := d.cards[d.position]
	d.position++
	return c
}

// Burn membuang satu kartu dari atas deck (sesuai aturan poker).
func (d *Deck) Burn() {
	d.Draw() // discard
}

// ==========================================================================
// Card rendering — format yang mobile-friendly untuk WhatsApp.
//
// WhatsApp TIDAK menggunakan monospace font, sehingga box-drawing
// characters (┌─┐│└─┘) akan berantakan. Kita pakai format sederhana
// dengan emoji yang terbaca rapi di semua device.
// ==========================================================================

// suitEmojis memetakan suit ke emoji berwarna yang lebih visual di WhatsApp.
var suitEmojis = map[Suit]string{
	Spade:   "♠️",
	Heart:   "♥️",
	Diamond: "♦️",
	Club:    "♣️",
}

// RenderCards mengembalikan representasi kartu untuk pesan WhatsApp.
// Format vertikal — satu kartu per baris agar mudah dibaca di HP.
//
// Contoh output untuk community cards [7♦, J♠, 2♣]:
//
//	7♦️ • J♠️ • 2♣️
//
// Untuk hole cards (2 kartu, via DM) format lebih detail:
//
//	♠️ A  (Ace)
//	♥️ K  (King)
func RenderCards(cards []Card) string {
	if len(cards) == 0 {
		return ""
	}

	// Untuk 2 kartu (hole cards), format vertikal detail
	if len(cards) <= 2 {
		var sb strings.Builder
		for i, c := range cards {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("%s %s  (%s)", suitEmojis[c.Suit], c.Rank.String(), c.Rank.Name()))
		}
		return sb.String()
	}

	// Untuk 3+ kartu (community cards), format horizontal compact
	parts := make([]string, len(cards))
	for i, c := range cards {
		parts[i] = c.Rank.String() + suitEmojis[c.Suit]
	}
	return strings.Join(parts, "  •  ")
}

// RenderCardsShort mengembalikan representasi singkat satu baris.
// Contoh: "🃏 A♠️ K♥️"
func RenderCardsShort(cards []Card) string {
	parts := make([]string, len(cards))
	for i, c := range cards {
		parts[i] = c.Rank.String() + suitEmojis[c.Suit]
	}
	return "🃏 " + strings.Join(parts, " ")
}

