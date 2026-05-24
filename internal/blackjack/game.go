package blackjack

import (
	"errors"
	"fmt"
)

// ==========================================================================
// BlackjackGame — core game engine for blackjack.
//
// Manages game state, players, dealer, and game flow.
// ==========================================================================

// GamePhase represents the current phase of the game
type GamePhase int

const (
	PhaseLobby GamePhase = iota
	PhaseDealing
	PhasePlayerTurns
	PhaseDealerTurn
	PhaseFinished
)

func (p GamePhase) String() string {
	switch p {
	case PhaseLobby:
		return "Lobby"
	case PhaseDealing:
		return "Dealing"
	case PhasePlayerTurns:
		return "Player Turns"
	case PhaseDealerTurn:
		return "Dealer Turn"
	case PhaseFinished:
		return "Finished"
	default:
		return "Unknown"
	}
}

// PlayerStatus represents the current status of a player
type PlayerStatus int

const (
	StatusActive PlayerStatus = iota
	StatusStand
	StatusBust
	StatusBlackjack
)

func (s PlayerStatus) String() string {
	switch s {
	case StatusActive:
		return "Active"
	case StatusStand:
		return "Stand"
	case StatusBust:
		return "Bust"
	case StatusBlackjack:
		return "Blackjack"
	default:
		return "Unknown"
	}
}

// BlackjackPlayer represents a player in the game
type BlackjackPlayer struct {
	Name      string
	JID       string
	Hand      *Hand
	Bet       int
	Chips     int          // Table balance / game chips (same as poker)
	Status    PlayerStatus
	IsDoubled bool // Track if player has doubled down
}

// BlackjackGame represents the game state
type BlackjackGame struct {
	Phase         GamePhase
	Deck          *Deck
	Players       []*BlackjackPlayer
	Dealer        *Hand
	CurrentPlayer int
	DealerUpCard  Card // The dealer's face-up card
}

const (
	MaxPlayers = 7 // Maximum players per game
)

var (
	ErrGameNotInLobby    = errors.New("game is not in lobby phase")
	ErrMaxPlayersReached = errors.New("maximum players reached")
	ErrPlayerExists      = errors.New("player already in game")
	ErrNoPlayers         = errors.New("no players in game")
	ErrNotEnoughPlayers  = errors.New("not enough players to start")
)

// ActionResult represents the result of a player action
type ActionResult struct {
	Valid      bool
	Message    string
	PlayerBust bool
	NextPlayer string
	DealerTurn bool
	RoundOver  bool
}

// PlayerInfo holds basic info of a player for round start reporting
type PlayerInfo struct {
	Name string
	JID  string
	Bet  int
}

// BetAdjustment records when sticky bet was lowered due to insufficient table chips.
type BetAdjustment struct {
	PlayerName string
	OldBet     int
	NewBet     int
}

// RoundStartInfo holds details of a newly started round
type RoundStartInfo struct {
	Players         []PlayerInfo
	DealerUpCard    Card
	BetAdjustments  []BetAdjustment
}

// DealerResult holds details of the dealer's final hand
type DealerResult struct {
	FinalValue int
	Cards      []Card
	IsBust     bool
}

// WinResult holds the outcome and payout for a single player
type WinResult struct {
	PlayerName string
	PlayerJID  string
	Outcome    string // "win", "lose", "push", "blackjack"
	Payout     int    // Amount to pay (0 for lose, bet for push, 2*bet for win, 2.5*bet for blackjack)
}

// NewBlackjackGame creates a new blackjack game
func NewBlackjackGame() *BlackjackGame {
	return &BlackjackGame{
		Phase:   PhaseLobby,
		Players: make([]*BlackjackPlayer, 0, MaxPlayers),
		Dealer:  NewHand(),
	}
}

// AddPlayer adds a player to the game (lobby, jeda antar-ronde, atau buy-in ulang saat meja kosong).
func (g *BlackjackGame) AddPlayer(name, jid string, chips int) error {
	if g.Phase != PhaseLobby && g.Phase != PhaseFinished {
		return ErrGameNotInLobby
	}

	if len(g.Players) >= MaxPlayers {
		return ErrMaxPlayersReached
	}

	// Check for duplicate player
	for _, p := range g.Players {
		if p.Name == name || p.JID == jid {
			return ErrPlayerExists
		}
	}

	// Sticky bet: taruhan awal mengikuti buy-in; tetap dipertahankan antar ronde via Reset().
	player := &BlackjackPlayer{
		Name:   name,
		JID:    jid,
		Hand:   NewHand(),
		Chips:  chips,
		Bet:    chips, // taruhan ronde pertama = nominal buy-in
		Status: StatusActive,
	}

	g.Players = append(g.Players, player)
	return nil
}

// SetPlayerBet updates the player's active bet size for the upcoming round.
func (g *BlackjackGame) SetPlayerBet(name string, bet int) error {
	p := g.GetPlayer(name)
	if p == nil {
		return fmt.Errorf("pemain %s tidak ditemukan", name)
	}
	if g.Phase != PhaseLobby && g.Phase != PhaseFinished {
		return fmt.Errorf("tidak bisa mengubah taruhan saat ronde sedang berjalan")
	}
	if bet <= 0 {
		return fmt.Errorf("taruhan harus angka positif")
	}
	if bet > p.Chips {
		return fmt.Errorf("saldo meja tidak cukup (tersedia: %d chip)", p.Chips)
	}
	p.Bet = bet
	return nil
}

// PlayerCount returns the number of players in the game
func (g *BlackjackGame) PlayerCount() int {
	return len(g.Players)
}

// GetPlayer returns a player by name
func (g *BlackjackGame) GetPlayer(name string) *BlackjackPlayer {
	for _, p := range g.Players {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// GetPlayerByJID returns a player by JID
func (g *BlackjackGame) GetPlayerByJID(jid string) *BlackjackPlayer {
	for _, p := range g.Players {
		if p.JID == jid {
			return p
		}
	}
	return nil
}

// RemovePlayer removes a player from the game
func (g *BlackjackGame) RemovePlayer(name string) bool {
	for i, p := range g.Players {
		if p.Name == name {
			// Remove player from slice
			g.Players = append(g.Players[:i], g.Players[i+1:]...)
			return true
		}
	}
	return false
}

// Reset resets the game state for a new round
func (g *BlackjackGame) Reset() {
	g.Phase = PhaseLobby
	g.Deck = nil
	g.Dealer.Clear()
	g.CurrentPlayer = 0
	g.DealerUpCard = Card{}

	// Clear player hands but keep players in game
	for _, p := range g.Players {
		p.Hand.Clear()
		p.Status = StatusActive
		if p.IsDoubled {
			p.Bet /= 2
		}
		p.IsDoubled = false
	}
}

// StartRound starts a new round of blackjack.
// It initializes the deck, shuffles, deals 2 cards to players & dealer,
// and checks for initial blackjacks.
func (g *BlackjackGame) StartRound() (*RoundStartInfo, error) {
	if g.Phase != PhaseLobby {
		return nil, ErrGameNotInLobby
	}

	if len(g.Players) == 0 {
		return nil, ErrNoPlayers
	}

	// Sticky bet: pertahankan taruhan ronde sebelumnya; kurangi otomatis jika saldo meja tidak cukup (all-in).
	var adjustments []BetAdjustment
	for _, p := range g.Players {
		originalBet := p.Bet
		if p.Bet <= 0 {
			if p.Chips > 0 {
				p.Bet = p.Chips
			} else {
				p.Bet = 0
			}
		} else if p.Chips < p.Bet {
			if p.Chips > 0 {
				p.Bet = p.Chips // all-in: taruhan diturunkan ke seluruh saldo meja
				adjustments = append(adjustments, BetAdjustment{
					PlayerName: p.Name,
					OldBet:     originalBet,
					NewBet:     p.Bet,
				})
			} else {
				p.Bet = 0
			}
		}
		p.Chips -= p.Bet
	}

	g.Phase = PhaseDealing

	// 1. Create and shuffle deck if not already injected (for testing)
	if g.Deck == nil {
		g.Deck = NewDeck()
		g.Deck.Shuffle()
	}

	// 2. Deal 2 cards to each player and dealer
	for i := 0; i < 2; i++ {
		for _, p := range g.Players {
			card := g.Deck.Draw()
			p.Hand.AddCard(card)
		}
		card := g.Deck.Draw()
		g.Dealer.AddCard(card)
	}

	// 3. Set DealerUpCard (first card is upcard)
	if len(g.Dealer.Cards) > 0 {
		g.DealerUpCard = g.Dealer.Cards[0]
	}

	// Prepare return info
	pInfos := make([]PlayerInfo, len(g.Players))
	for i, p := range g.Players {
		pInfos[i] = PlayerInfo{
			Name: p.Name,
			JID:  p.JID,
			Bet:  p.Bet,
		}
	}

	info := &RoundStartInfo{
		Players:        pInfos,
		DealerUpCard:   g.DealerUpCard,
		BetAdjustments: adjustments,
	}

	// 4. Check for dealer blackjack
	if g.Dealer.IsBlackjack() {
		// Immediately check all players for blackjack to determine push/lose
		for _, p := range g.Players {
			if p.Hand.IsBlackjack() {
				p.Status = StatusBlackjack
			} else {
				p.Status = StatusStand
			}
		}
		g.Phase = PhaseFinished
		return info, nil
	}

	// 5. If dealer doesn't have blackjack, check if players have natural blackjack
	allBlackjack := true
	for _, p := range g.Players {
		if p.Hand.IsBlackjack() {
			p.Status = StatusBlackjack
		} else {
			p.Status = StatusActive
			allBlackjack = false
		}
	}

	if allBlackjack {
		// Everyone has blackjack, straight to dealer turn
		g.Phase = PhaseDealerTurn
	} else {
		g.Phase = PhasePlayerTurns
		// Find first active player
		g.CurrentPlayer = 0
		for g.CurrentPlayer < len(g.Players) && g.Players[g.CurrentPlayer].Status != StatusActive {
			g.CurrentPlayer++
		}
	}

	return info, nil
}

// Hit deals one card to the active player.
func (g *BlackjackGame) Hit(playerName string) (*ActionResult, error) {
	if g.Phase != PhasePlayerTurns {
		return nil, errors.New("game is not in player turns phase")
	}

	if g.CurrentPlayer >= len(g.Players) {
		return nil, errors.New("invalid player turn index")
	}

	currPlayer := g.Players[g.CurrentPlayer]
	if currPlayer.Name != playerName {
		return nil, fmt.Errorf("not %s's turn (it is %s's turn)", playerName, currPlayer.Name)
	}

	if currPlayer.Status != StatusActive {
		return nil, errors.New("player is not active")
	}

	// Deal 1 card
	card := g.Deck.Draw()
	currPlayer.Hand.AddCard(card)

	res := &ActionResult{
		Valid: true,
	}

	if currPlayer.Hand.IsBust() {
		currPlayer.Status = StatusBust
		res.PlayerBust = true
		res.Message = fmt.Sprintf("Bust! %s's hand value is %d.", currPlayer.Name, currPlayer.Hand.Value())
		g.advanceTurn(res)
	} else if currPlayer.Hand.Value() == 21 {
		currPlayer.Status = StatusStand
		res.Message = fmt.Sprintf("%s has 21!", currPlayer.Name)
		g.advanceTurn(res)
	} else {
		res.Message = fmt.Sprintf("%s hits and receives %s. Total: %d.", currPlayer.Name, card.String(), currPlayer.Hand.Value())
		res.NextPlayer = currPlayer.Name
	}

	return res, nil
}

// Stand ends the active player's turn.
func (g *BlackjackGame) Stand(playerName string) (*ActionResult, error) {
	if g.Phase != PhasePlayerTurns {
		return nil, errors.New("game is not in player turns phase")
	}

	if g.CurrentPlayer >= len(g.Players) {
		return nil, errors.New("invalid player turn index")
	}

	currPlayer := g.Players[g.CurrentPlayer]
	if currPlayer.Name != playerName {
		return nil, fmt.Errorf("not %s's turn (it is %s's turn)", playerName, currPlayer.Name)
	}

	if currPlayer.Status != StatusActive {
		return nil, errors.New("player is not active")
	}

	currPlayer.Status = StatusStand

	res := &ActionResult{
		Valid:   true,
		Message: fmt.Sprintf("%s stands with %d.", currPlayer.Name, currPlayer.Hand.Value()),
	}

	g.advanceTurn(res)
	return res, nil
}

// DoubleDown doubles the player's bet, deals exactly one card, and stands.
func (g *BlackjackGame) DoubleDown(playerName string) (*ActionResult, error) {
	if g.Phase != PhasePlayerTurns {
		return nil, errors.New("game is not in player turns phase")
	}

	if g.CurrentPlayer >= len(g.Players) {
		return nil, errors.New("invalid player turn index")
	}

	currPlayer := g.Players[g.CurrentPlayer]
	if currPlayer.Name != playerName {
		return nil, fmt.Errorf("not %s's turn (it is %s's turn)", playerName, currPlayer.Name)
	}

	if currPlayer.Status != StatusActive {
		return nil, errors.New("player is not active")
	}

	if len(currPlayer.Hand.Cards) != 2 {
		return nil, errors.New("can only double down with exactly 2 cards")
	}

	originalBet := currPlayer.Bet
	if currPlayer.Chips < originalBet {
		return nil, fmt.Errorf("saldo meja tidak cukup untuk double down (butuh %d, tersedia %d chip)", originalBet, currPlayer.Chips)
	}

	currPlayer.Chips -= originalBet
	currPlayer.IsDoubled = true
	currPlayer.Bet *= 2

	// Deal exactly 1 card
	card := g.Deck.Draw()
	currPlayer.Hand.AddCard(card)

	res := &ActionResult{
		Valid: true,
	}

	if currPlayer.Hand.IsBust() {
		currPlayer.Status = StatusBust
		res.PlayerBust = true
		res.Message = fmt.Sprintf("Double Down! %s receives %s and Busts with %d!", currPlayer.Name, card.String(), currPlayer.Hand.Value())
	} else {
		currPlayer.Status = StatusStand
		res.Message = fmt.Sprintf("Double Down! %s receives %s. Final total: %d.", currPlayer.Name, card.String(), currPlayer.Hand.Value())
	}

	g.advanceTurn(res)
	return res, nil
}

// PlayDealerTurn automates the dealer's turn.
// The dealer must hit on 16 or lower, and stand on 17 or higher.
func (g *BlackjackGame) PlayDealerTurn() *DealerResult {
	g.Phase = PhaseDealerTurn

	// Dealer plays according to rules
	for g.Dealer.Value() <= 16 {
		card := g.Deck.Draw()
		g.Dealer.AddCard(card)
	}

	isBust := g.Dealer.IsBust()

	return &DealerResult{
		FinalValue: g.Dealer.Value(),
		Cards:      g.Dealer.Cards,
		IsBust:     isBust,
	}
}

// DetermineWinners compares players' hands against the dealer's hand
// and returns the results.
func (g *BlackjackGame) DetermineWinners() []WinResult {
	g.Phase = PhaseFinished
	results := make([]WinResult, 0, len(g.Players))

	dealerVal := g.Dealer.Value()
	dealerBust := g.Dealer.IsBust()
	dealerBJ := g.Dealer.IsBlackjack()

	for _, p := range g.Players {
		var outcome string
		var payout float64

		playerVal := p.Hand.Value()
		playerBust := p.Hand.IsBust()
		playerBJ := p.Hand.IsBlackjack()

		if playerBust {
			outcome = "lose"
			payout = 0
		} else if playerBJ {
			if dealerBJ {
				outcome = "push"
				payout = float64(p.Bet)
			} else {
				outcome = "blackjack"
				payout = 2.5 * float64(p.Bet)
			}
		} else if dealerBJ {
			outcome = "lose"
			payout = 0
		} else if dealerBust {
			outcome = "win"
			payout = float64(p.Bet) * 2
		} else if playerVal > dealerVal {
			outcome = "win"
			payout = float64(p.Bet) * 2
		} else if playerVal == dealerVal {
			outcome = "push"
			payout = float64(p.Bet)
		} else {
			outcome = "lose"
			payout = 0
		}

		results = append(results, WinResult{
			PlayerName: p.Name,
			PlayerJID:  p.JID,
			Outcome:    outcome,
			Payout:     int(payout),
		})
	}

	return results
}

func (g *BlackjackGame) advanceTurn(res *ActionResult) {
	g.CurrentPlayer++
	// Find next active player
	for g.CurrentPlayer < len(g.Players) && g.Players[g.CurrentPlayer].Status != StatusActive {
		g.CurrentPlayer++
	}

	if g.CurrentPlayer >= len(g.Players) {
		g.Phase = PhaseDealerTurn
		res.DealerTurn = true
		res.RoundOver = false
	} else {
		res.NextPlayer = g.Players[g.CurrentPlayer].Name
	}
}
