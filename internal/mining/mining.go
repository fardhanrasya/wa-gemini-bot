package mining

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"wa-gemini-bot/internal/economy"
)

var (
	ErrMaxRigsReached     = errors.New("kamu sudah mencapai batas maksimal 3 rig aktif")
	ErrInsufficientChips  = errors.New("saldo chip kamu tidak mencukupi untuk membeli rig ini")
	ErrLowRank            = errors.New("pangkat kamu belum memenuhi syarat untuk membeli rig tier ini")
	ErrInvalidTier        = errors.New("tier rig tidak valid")
	ErrSessionExpired     = errors.New("sesi sudah kedaluwarsa, silakan minta tautan login baru di WhatsApp")
	ErrInvalidToken       = errors.New("token login tidak valid atau sudah kedaluwarsa")
)

// RigTier mendefinisikan properti rig untuk setiap tier penambangan.
type RigTier struct {
	Name           string
	Cost           int
	BaseSpeed      int // chip per jam
	MaxDurability  int // kapasitas total produksi sebelum rusak
	RequiredRank   string
	RequiredMinBal int
	Description    string
}

// Daftar tier rig yang tersedia di game.
var Tiers = map[string]RigTier{
	"basic": {
		Name:           "Basic Drill",
		Cost:           1000,
		BaseSpeed:      10,
		MaxDurability:  3000, // untung bersih +2000 chip
		RequiredRank:   "Peasant",
		RequiredMinBal: 0,
		Description:    "Alat tambang standar yang murah dan ramah pemula.",
	},
	"advanced": {
		Name:           "CUDA Rig",
		Cost:           5000,
		BaseSpeed:      35,
		MaxDurability:  12000, // untung bersih +7000 chip
		RequiredRank:   "Levy",
		RequiredMinBal: 5000,
		Description:    "Rig bertenaga GPU CUDA dengan hasil yang lebih optimal.",
	},
	"quantum": {
		Name:           "Quantum Core",
		Cost:           20000,
		BaseSpeed:      100,
		MaxDurability:  50000, // untung bersih +30000 chip
		RequiredRank:   "Mercenary",
		RequiredMinBal: 20000,
		Description:    "Teknologi penambangan kuantum tercanggih untuk para profesional.",
	},
}

// Rig merepresentasikan satu instance alat tambang milik pemain di database.
type Rig struct {
	ID           int       `json:"id"`
	JID          string    `json:"jid"`
	Tier         string    `json:"tier"`
	Efficiency   float64   `json:"efficiency"`
	ChipsMined   int       `json:"chips_mined"`
	MaxDurability int       `json:"max_durability"`
	LastFuelTime time.Time `json:"last_fuel_time"`
	CreatedAt    time.Time `json:"created_at"`
}

// RigStatus merepresentasikan state real-time dari rig hasil kalkulasi lazy evaluation.
type RigStatus struct {
	Rig
	PendingChips      int     `json:"pending_chips"`
	RemainingLifespan int     `json:"remaining_lifespan"`
	FuelPercent       float64 `json:"fuel_percent"`
	EffectiveSpeed    float64 `json:"effective_speed"`
	IsBroken          bool    `json:"is_broken"`
}

// MiningService mengelola semua logika bisnis pertambangan chip dan autentikasi dashboard.
type MiningService struct {
	db  *sql.DB
	eco *economy.EconomyService
}

// NewMiningService membuat instance MiningService baru.
func NewMiningService(eco *economy.EconomyService) *MiningService {
	return &MiningService{
		db:  eco.DB(),
		eco: eco,
	}
}

// ==========================================================================
// 1. GAME LOGIC (LAZY EVALUATION)
// ==========================================================================

// GetActiveRigs mengambil semua rig milik pemain dan menghitung status real-time mereka.
func (s *MiningService) GetActiveRigs(jid string) ([]RigStatus, error) {
	rows, err := s.db.Query(
		"SELECT id, jid, tier, efficiency, chips_mined, max_durability, last_fuel_time, created_at FROM mining_rigs WHERE jid = ? ORDER BY id ASC",
		jid,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal query mining_rigs: %w", err)
	}
	defer rows.Close()

	var statuses []RigStatus
	for rows.Next() {
		var r Rig
		var lastFuelStr, createdStr string
		err := rows.Scan(&r.ID, &r.JID, &r.Tier, &r.Efficiency, &r.ChipsMined, &r.MaxDurability, &lastFuelStr, &createdStr)
		if err != nil {
			return nil, err
		}

		r.LastFuelTime, _ = time.Parse("2006-01-02 15:04:05", lastFuelStr)
		if r.LastFuelTime.IsZero() {
			// fallback jika SQLite datetime format sedikit berbeda
			r.LastFuelTime, _ = time.Parse(time.RFC3339, lastFuelStr)
		}
		r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)

		statuses = append(statuses, s.calculateRigStatus(r))
	}

	return statuses, nil
}

// GetAllPlayersRigs mengambil status rig publik milik semua pemain (read-only untuk dashboard umum).
type PlayerPublicMining struct {
	Name      string      `json:"name"`
	JID       string      `json:"jid"`
	Balance   int         `json:"balance"`
	RankName  string      `json:"rank_name"`
	RankEmoji string      `json:"rank_emoji"`
	Rigs      []RigStatus `json:"rigs"`
}

func (s *MiningService) GetAllPlayersRigs() ([]PlayerPublicMining, error) {
	// Ambil semua pengguna dari database economy
	rows, err := s.db.Query("SELECT jid, name, balance FROM users ORDER BY balance DESC LIMIT 50")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []PlayerPublicMining
	for rows.Next() {
		var p PlayerPublicMining
		err := rows.Scan(&p.JID, &p.Name, &p.Balance)
		if err != nil {
			return nil, err
		}

		if p.Name == "" {
			p.Name = "Unknown"
		}

		rank := economy.GetRankByBalance(p.Balance)
		p.RankName = rank.Name
		p.RankEmoji = rank.Emoji

		// Ambil rig aktif milik user ini
		rigs, errRigs := s.GetActiveRigs(p.JID)
		if errRigs != nil {
			continue
		}

		// Hanya tampilkan di public dashboard jika user memiliki minimal 1 rig
		if len(rigs) > 0 {
			p.Rigs = rigs
			list = append(list, p)
		}
	}

	return list, nil
}

// calculateRigStatus menghitung state pending koin, sisa fuel, dan keausan (durability) rig secara Lazy Evaluation.
func (s *MiningService) calculateRigStatus(r Rig) RigStatus {
	tierProps, ok := Tiers[r.Tier]
	if !ok {
		return RigStatus{Rig: r, IsBroken: true}
	}

	status := RigStatus{
		Rig:               r,
		RemainingLifespan: r.MaxDurability - r.ChipsMined,
		EffectiveSpeed:    float64(tierProps.BaseSpeed) * r.Efficiency,
	}

	if status.RemainingLifespan <= 0 {
		status.RemainingLifespan = 0
		status.IsBroken = true
		status.FuelPercent = 0
		status.PendingChips = 0
		return status
	}

	// 1. Hitung selisih waktu sejak klaim/refuel terakhir
	elapsed := time.Now().UTC().Sub(r.LastFuelTime)
	
	// 2. Batasi waktu maksimal 24 jam (karena tangki bahan bakar habis)
	fuelLimit := 24 * time.Hour
	if elapsed > fuelLimit {
		elapsed = fuelLimit
	}

	// 3. Hitung persentase sisa bahan bakar
	status.FuelPercent = ((fuelLimit - elapsed).Hours() / 24.0) * 100.0
	if status.FuelPercent < 0 {
		status.FuelPercent = 0
	}

	// 4. Hitung chip yang terproduksi (kecepatan per jam * selisih jam)
	generated := elapsed.Hours() * status.EffectiveSpeed
	pending := int(generated)

	// 5. Batasi perolehan agar tidak melebihi sisa durabilitas rig
	if pending > status.RemainingLifespan {
		pending = status.RemainingLifespan
	}

	status.PendingChips = pending
	return status
}

// BuyRig memproses transaksi pembelian rig baru.
func (s *MiningService) BuyRig(jid string, tier string) error {
	tierProps, ok := Tiers[tier]
	if !ok {
		return ErrInvalidTier
	}

	// 1. Ambil data profil ekonomi user
	user, err := s.eco.GetUser(jid)
	if err != nil {
		return fmt.Errorf("gagal mendapatkan profil ekonomi user: %w", err)
	}

	// 2. Cek kecukupan pangkat
	hasRequiredRank := false
	for _, r := range economy.Ranks {
		if r.Name == tierProps.RequiredRank {
			if user.Balance >= r.MinChips {
				hasRequiredRank = true
			}
			break
		}
	}
	// Fallback check jika rank minimal Peasant
	if tierProps.RequiredMinBal == 0 {
		hasRequiredRank = true
	} else if user.Balance < tierProps.RequiredMinBal {
		hasRequiredRank = false
	}

	if !hasRequiredRank {
		return fmt.Errorf("%w (minimal %s / saldo >= %d chip)", ErrLowRank, tierProps.RequiredRank, tierProps.RequiredMinBal)
	}

	// 3. Ambil daftar rig aktif saat ini
	rigs, err := s.GetActiveRigs(jid)
	if err != nil {
		return err
	}

	// Filter rig yang belum rusak
	var activeCount int
	for _, rg := range rigs {
		if !rg.IsBroken {
			activeCount++
		}
	}

	if activeCount >= 3 {
		return ErrMaxRigsReached
	}

	// 4. Tentukan tingkat efisiensi rig baru (diminishing returns)
	// Rig 1 = 100% (1.0), Rig 2 = 75% (0.75), Rig 3 = 50% (0.50)
	var efficiency float64
	switch activeCount {
	case 0:
		efficiency = 1.0
	case 1:
		efficiency = 0.75
	default:
		efficiency = 0.50
	}

	// 5. Potong saldo chip secara atomik via SubtractBalance
	txRef := fmt.Sprintf("buy_rig_%s_eff_%.2f", tier, efficiency)
	if err := s.eco.SubtractBalance(jid, tierProps.Cost, "mining_buy_rig", txRef); err != nil {
		if errors.Is(err, economy.ErrInsufficientFunds) {
			return ErrInsufficientChips
		}
		return fmt.Errorf("gagal memotong saldo: %w", err)
	}

	// 6. Masukkan data rig baru ke database
	_, errInsert := s.db.Exec(
		"INSERT INTO mining_rigs (jid, tier, efficiency, max_durability, last_fuel_time) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)",
		jid, tier, efficiency, tierProps.MaxDurability,
	)
	if errInsert != nil {
		// Log error kritis dan usahakan refund jika insert gagal
		log.Printf("[MINING] ❌ CRITICAL: Gagal insert rig setelah potong saldo! Refund saldo user %s...", jid)
		_ = s.eco.AddBalance(jid, tierProps.Cost, "mining_refund_failed_buy", txRef)
		return fmt.Errorf("gagal mencatat rig baru di database: %w", errInsert)
	}

	log.Printf("[MINING] ⚙️ PURCHASE: %s membeli %s (Efisiensi: %.2f) seharga %d chip", jid, tierProps.Name, efficiency, tierProps.Cost)
	return nil
}

// ClaimAndRefuel mencairkan koin dari semua rig aktif pemain, menambahkan ke saldo utama, dan mereset bahan bakar.
func (s *MiningService) ClaimAndRefuel(jid string) (int, error) {
	rigs, err := s.GetActiveRigs(jid)
	if err != nil {
		return 0, err
	}

	totalPending := 0
	claimedRigsCount := 0

	tx, errTx := s.db.Begin()
	if errTx != nil {
		return 0, fmt.Errorf("gagal memulai transaksi klaim: %w", errTx)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	nowStr := now.Format("2006-01-02 15:04:05")

	for _, rg := range rigs {
		// Abaikan rig yang sudah rusak total dan tidak ada pending chips
		if rg.IsBroken && rg.PendingChips == 0 {
			continue
		}

		if rg.PendingChips > 0 {
			totalPending += rg.PendingChips
			newChipsMined := rg.ChipsMined + rg.PendingChips

			// Update total chip yang ditambang dan reset waktu bahan bakar ke saat ini
			_, errUpdate := tx.Exec(
				"UPDATE mining_rigs SET chips_mined = ?, last_fuel_time = ? WHERE id = ?",
				newChipsMined, nowStr, rg.ID,
			)
			if errUpdate != nil {
				return 0, fmt.Errorf("gagal mengupdate rig ID %d: %w", rg.ID, errUpdate)
			}
			claimedRigsCount++
		} else {
			// Jika tidak ada pending chips, hanya reset bahan bakar agar penuh kembali
			_, errRefuelOnly := tx.Exec(
				"UPDATE mining_rigs SET last_fuel_time = ? WHERE id = ? AND chips_mined < max_durability",
				nowStr, rg.ID,
			)
			if errRefuelOnly != nil {
				return 0, fmt.Errorf("gagal mereset bahan bakar rig ID %d: %w", rg.ID, errRefuelOnly)
			}
		}
	}

	if errCommit := tx.Commit(); errCommit != nil {
		return 0, fmt.Errorf("gagal commit transaksi klaim: %w", errCommit)
	}

	// Tambahkan total chip terklaim ke dompet ekonomi pemain secara langsung
	if totalPending > 0 {
		ref := fmt.Sprintf("claim_rigs_count_%d", claimedRigsCount)
		if errAdd := s.eco.AddBalance(jid, totalPending, "mining_claim", ref); errAdd != nil {
			return 0, fmt.Errorf("gagal menyetorkan chip hasil tambang ke dompet: %w", errAdd)
		}
		log.Printf("[MINING] ⚡ CLAIM: %s mencairkan *%d* chip dari %d rig aktif & refuel baterai.", jid, totalPending, claimedRigsCount)
	} else {
		log.Printf("[MINING] 🔋 REFUEL ONLY: %s mereset bahan bakar tanpa pencairan koin.", jid)
	}

	return totalPending, nil
}

// ==========================================================================
// 2. AUTHENTICATION LOGIC (MAGIC LINKS & COOKIE SESSIONS)
// ==========================================================================

func generateSecureToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateLoginToken membuat token satu-kali login (berlaku 1 jam).
func (s *MiningService) CreateLoginToken(jid string) (string, error) {
	token := generateSecureToken()
	expiresAt := time.Now().UTC().Add(1 * time.Hour).Format("2006-01-02 15:04:05")

	// Bersihkan token lama milik JID ini agar tidak menumpuk sampah
	_, _ = s.db.Exec("DELETE FROM web_login_tokens WHERE jid = ?", jid)

	_, err := s.db.Exec(
		"INSERT INTO web_login_tokens (token, jid, expires_at) VALUES (?, ?, ?)",
		token, jid, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan token login: %w", err)
	}

	log.Printf("[MINING] 🔑 TOKEN GENERATED: Token satu-kali dibuat untuk %s", jid)
	return token, nil
}

// VerifyLoginToken mencocokkan token login satu-kali. Jika valid, return JID dan hapus token tersebut (single use).
func (s *MiningService) VerifyLoginToken(token string) (string, error) {
	var jid, expiresStr string
	err := s.db.QueryRow(
		"SELECT jid, expires_at FROM web_login_tokens WHERE token = ?",
		token,
	).Scan(&jid, &expiresStr)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidToken
		}
		return "", err
	}

	expiresAt, errParse := time.Parse("2006-01-02 15:04:05", expiresStr)
	if errParse != nil {
		expiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	}

	// Cek kedaluwarsa
	if time.Now().After(expiresAt) {
		_, _ = s.db.Exec("DELETE FROM web_login_tokens WHERE token = ?", token)
		return "", ErrInvalidToken
	}

	// Hapus token setelah digunakan sekali (single use)
	_, _ = s.db.Exec("DELETE FROM web_login_tokens WHERE token = ?", token)
	log.Printf("[MINING] 🔓 TOKEN VERIFIED: Token berhasil diverifikasi untuk %s", jid)
	return jid, nil
}

// CreateSession membuat cookie session ID aktif (berlaku 7 hari).
func (s *MiningService) CreateSession(jid string) (string, error) {
	sessionID := generateSecureToken()
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05")

	_, err := s.db.Exec(
		"INSERT INTO web_sessions (session_id, jid, expires_at) VALUES (?, ?, ?)",
		sessionID, jid, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan sesi baru: %w", err)
	}

	log.Printf("[MINING] 🔌 SESSION CREATED: Sesi %s terdaftar untuk JID %s", sessionID[:6], jid)
	return sessionID, nil
}

// VerifySession memvalidasi session ID dari Cookie. Jika valid, return JID pemain.
func (s *MiningService) VerifySession(sessionID string) (string, error) {
	var jid, expiresStr string
	err := s.db.QueryRow(
		"SELECT jid, expires_at FROM web_sessions WHERE session_id = ?",
		sessionID,
	).Scan(&jid, &expiresStr)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrSessionExpired
		}
		return "", err
	}

	expiresAt, errParse := time.Parse("2006-01-02 15:04:05", expiresStr)
	if errParse != nil {
		expiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	}

	// Cek kedaluwarsa
	if time.Now().After(expiresAt) {
		_, _ = s.db.Exec("DELETE FROM web_sessions WHERE session_id = ?", sessionID)
		return "", ErrSessionExpired
	}

	return jid, nil
}

// DestroySession menghapus sesi aktif (Logout).
func (s *MiningService) DestroySession(sessionID string) {
	_, _ = s.db.Exec("DELETE FROM web_sessions WHERE session_id = ?", sessionID)
	log.Printf("[MINING] 🔒 SESSION DESTROYED: Sesi %s dihapus.", sessionID[:6])
}
