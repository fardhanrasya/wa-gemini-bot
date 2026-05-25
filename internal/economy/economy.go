package economy

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	InitialBalance       = 5000                   // Modal awal yang diberikan ke pemain baru
	DealerJID            = "dealer@abdul.bot" // JID khusus untuk dealer bot
	DealerName           = "🎰 Dealer Bot"
	DealerInitialBalance = 1000000000 // 1 miliar chip untuk dealer
)

var (
	ErrInsufficientFunds = errors.New("saldo tidak mencukupi")
	ErrSelfTransfer      = errors.New("tidak bisa transfer ke diri sendiri")
	ErrInvalidAmount     = errors.New("jumlah tidak valid")
)

// EconomyService mengelola mata uang/chip pemain secara persisten
// menggunakan database SQLite.
//
// Setiap mutasi saldo dicatat secara atomik di tabel transaction_logs,
// sehingga ada audit trail lengkap untuk debugging dan recovery.
type EconomyService struct {
	db *sql.DB
	mu sync.RWMutex
}

// User merepresentasikan data ekonomi seorang pemain.
type User struct {
	JID     string
	Name    string
	Balance int
}

// TransactionLog merepresentasikan satu record log transaksi.
type TransactionLog struct {
	ID           int
	JID          string
	Amount       int
	BalanceAfter int
	Type         string
	Reference    string
	CreatedAt    time.Time
}

// NewEconomyService membuat atau membuka koneksi ke database SQLite.
// Schema migration bersifat additive — tabel baru ditambahkan tanpa
// mengubah tabel yang sudah ada, sehingga data lama aman.
func NewEconomyService(dbPath string) (*EconomyService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database economy: %w", err)
	}

	// Buat tabel users jika belum ada (schema asli, tidak diubah)
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		jid TEXT PRIMARY KEY,
		name TEXT DEFAULT '',
		balance INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(usersTable); err != nil {
		return nil, fmt.Errorf("gagal membuat tabel users: %w", err)
	}

	// Tabel transaction_logs — audit trail untuk setiap mutasi saldo.
	// Setiap row merepresentasikan satu perubahan balance:
	//   - amount: positif = masuk, negatif = keluar
	//   - balance_after: saldo setelah transaksi (untuk verifikasi konsistensi)
	//   - type: kategori transaksi (poker_buyin, trivia_reward, transfer_out, dll)
	//   - reference: konteks tambahan (misal group JID, round number)
	txLogsTable := `
	CREATE TABLE IF NOT EXISTS transaction_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL,
		amount INTEGER NOT NULL,
		balance_after INTEGER NOT NULL,
		type TEXT NOT NULL,
		reference TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(txLogsTable); err != nil {
		return nil, fmt.Errorf("gagal membuat tabel transaction_logs: %w", err)
	}

	// Index untuk query by JID (riwayat transaksi per user)
	indexQuery := `CREATE INDEX IF NOT EXISTS idx_tx_logs_jid ON transaction_logs(jid);`
	if _, err := db.Exec(indexQuery); err != nil {
		return nil, fmt.Errorf("gagal membuat index transaction_logs: %w", err)
	}

	// Migrasi database: tambahkan kolom name_is_custom secara aman jika belum ada
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN name_is_custom INTEGER DEFAULT 0;")

	// ==========================================
	// MIGRASI TABEL MINING WEB DASHBOARD
	// ==========================================
	
	// Tabel mining_rigs
	miningRigsTable := `
	CREATE TABLE IF NOT EXISTS mining_rigs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		jid TEXT NOT NULL,
		tier TEXT NOT NULL,
		efficiency REAL NOT NULL,
		chips_mined INTEGER DEFAULT 0,
		max_durability INTEGER NOT NULL,
		last_fuel_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(miningRigsTable); err != nil {
		return nil, fmt.Errorf("gagal membuat tabel mining_rigs: %w", err)
	}

	// Index untuk query cepat by JID
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_mining_rigs_jid ON mining_rigs(jid);`); err != nil {
		return nil, fmt.Errorf("gagal membuat index mining_rigs: %w", err)
	}

	// Tabel web_login_tokens
	webLoginTokensTable := `
	CREATE TABLE IF NOT EXISTS web_login_tokens (
		token TEXT PRIMARY KEY,
		jid TEXT NOT NULL,
		expires_at DATETIME NOT NULL
	);
	`
	if _, err := db.Exec(webLoginTokensTable); err != nil {
		return nil, fmt.Errorf("gagal membuat tabel web_login_tokens: %w", err)
	}

	// Tabel web_sessions
	webSessionsTable := `
	CREATE TABLE IF NOT EXISTS web_sessions (
		session_id TEXT PRIMARY KEY,
		jid TEXT NOT NULL,
		expires_at DATETIME NOT NULL
	);
	`
	if _, err := db.Exec(webSessionsTable); err != nil {
		return nil, fmt.Errorf("gagal membuat tabel web_sessions: %w", err)
	}

	service := &EconomyService{
		db: db,
	}

	// Initialize dealer account untuk blackjack
	if err := service.InitializeDealerAccount(); err != nil {
		return nil, fmt.Errorf("gagal inisialisasi akun dealer: %w", err)
	}

	return service, nil
}

// Close menutup koneksi database.
func (s *EconomyService) Close() error {
	return s.db.Close()
}

// InitializeDealerAccount membuat akun khusus untuk dealer bot blackjack.
// Akun ini digunakan untuk tracking keluar-masuk chip melalui ledger.
// Dealer memiliki saldo awal yang sangat besar untuk memastikan selalu bisa membayar.
func (s *EconomyService) InitializeDealerAccount() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cek apakah dealer sudah ada
	var balance int
	err := s.db.QueryRow("SELECT balance FROM users WHERE jid = ?", DealerJID).Scan(&balance)
	if err == nil {
		// Dealer sudah ada, tidak perlu inisialisasi ulang
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi dealer init: %w", err)
	}
	defer tx.Rollback()

	// Buat akun dealer dengan nama kustom (name_is_custom = 1)
	_, err = tx.Exec(
		"INSERT INTO users (jid, name, balance, name_is_custom) VALUES (?, ?, ?, 1)",
		DealerJID, DealerName, DealerInitialBalance,
	)
	if err != nil {
		return fmt.Errorf("gagal membuat akun dealer: %w", err)
	}

	// Log transaksi inisialisasi dealer
	if err := logTransaction(tx, DealerJID, DealerInitialBalance, DealerInitialBalance, "dealer_init", "system"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit dealer init: %w", err)
	}

	log.Printf("[ECONOMY] Akun dealer diinisialisasi dengan saldo %d", DealerInitialBalance)
	return nil
}

// UpdateName mengupdate nama tampilan pengguna untuk keperluan leaderboard.
// Hanya mengupdate jika pengguna tidak menggunakan nama kustom.
func (s *EconomyService) UpdateName(jid, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var isCustom int
	err := s.db.QueryRow("SELECT name_is_custom FROM users WHERE jid = ?", jid).Scan(&isCustom)
	if err == nil && isCustom == 1 {
		return nil // Jangan overwrite nama kustom dengan display name WA
	}

	_, err = s.db.Exec("UPDATE users SET name = ? WHERE jid = ?", name, jid)
	return err
}

// SetCustomName menyimpan nama kustom pilihan pengguna dan mengunci dari auto-update.
func (s *EconomyService) SetCustomName(jid, newName string) error {
	// Pastikan user terdaftar dulu di database.
	// Kita panggil GetBalance tanpa memegang Lock utama karena GetBalance mengelola lock-nya sendiri.
	_, err := s.GetBalance(jid)
	if err != nil {
		return fmt.Errorf("gagal mendaftarkan user untuk nama kustom: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err = s.db.Exec("UPDATE users SET name = ?, name_is_custom = 1 WHERE jid = ?", newName, jid)
	return err
}

// ResetCustomName mengembalikan nama pengguna agar ter-update otomatis mengikuti profil WA.
func (s *EconomyService) ResetCustomName(jid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("UPDATE users SET name_is_custom = 0 WHERE jid = ?", jid)
	return err
}

// GetLeaderboard mengembalikan daftar pemain dengan saldo tertinggi.
func (s *EconomyService) GetLeaderboard(limit int) ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT jid, name, balance FROM users WHERE jid != ? ORDER BY balance DESC LIMIT ?", DealerJID, limit)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil leaderboard: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.JID, &u.Name, &u.Balance); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// GetUser mengambil data lengkap profil ekonomi seorang pemain.
func (s *EconomyService) GetUser(jid string) (User, error) {
	// Pastikan user terdaftar
	_, err := s.GetBalance(jid)
	if err != nil {
		return User{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var u User
	u.JID = jid
	err = s.db.QueryRow("SELECT name, balance FROM users WHERE jid = ?", jid).Scan(&u.Name, &u.Balance)
	if err != nil {
		return User{}, fmt.Errorf("gagal mengambil profil user: %w", err)
	}
	return u, nil
}

// GetBalance mengembalikan saldo pengguna saat ini.
// Jika pengguna belum terdaftar di database, maka akan otomatis dibuatkan
// dan diberikan modal awal (InitialBalance).
func (s *EconomyService) GetBalance(jid string) (int, error) {
	s.mu.RLock()
	var balance int
	err := s.db.QueryRow("SELECT balance FROM users WHERE jid = ?", jid).Scan(&balance)
	s.mu.RUnlock()

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Otomatis daftar dengan modal awal jika belum ada
			if err := s.registerNewUser(jid); err != nil {
				return 0, err
			}
			return InitialBalance, nil
		}
		return 0, fmt.Errorf("gagal mengambil saldo: %w", err)
	}
	return balance, nil
}

func (s *EconomyService) registerNewUser(jid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check to prevent race conditions during insertion
	var balance int
	err := s.db.QueryRow("SELECT balance FROM users WHERE jid = ?", jid).Scan(&balance)
	if err == nil {
		return nil // Already exists
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi register: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO users (jid, balance) VALUES (?, ?)", jid, InitialBalance)
	if err != nil {
		return fmt.Errorf("gagal register user baru: %w", err)
	}

	// Log transaksi modal awal
	if err := logTransaction(tx, jid, InitialBalance, InitialBalance, "initial", "new_user_registration"); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit register: %w", err)
	}

	log.Printf("[ECONOMY] User baru %s mendapatkan modal awal %d", jid, InitialBalance)
	return nil
}

// AddBalance menambahkan sejumlah saldo ke pengguna.
//
// txType mengidentifikasi sumber penambahan (misal: "poker_refund", "poker_win",
// "trivia_reward"). reference memberikan konteks tambahan (misal: group JID).
// Keduanya dicatat di transaction_logs untuk audit trail.
func (s *EconomyService) AddBalance(jid string, amount int, txType, reference string) error {
	if amount <= 0 {
		return nil
	}

	// Pastikan user terdaftar
	_, err := s.GetBalance(jid)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi add: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE users SET balance = balance + ? WHERE jid = ?", amount, jid)
	if err != nil {
		return fmt.Errorf("gagal menambah saldo: %w", err)
	}

	// Baca saldo setelah update untuk dicatat di log
	var balanceAfter int
	if err := tx.QueryRow("SELECT balance FROM users WHERE jid = ?", jid).Scan(&balanceAfter); err != nil {
		return fmt.Errorf("gagal baca saldo setelah add: %w", err)
	}

	if err := logTransaction(tx, jid, amount, balanceAfter, txType, reference); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit add balance: %w", err)
	}

	log.Printf("[ECONOMY] %s +%d (%s: %s) → saldo: %d", jid, amount, txType, reference, balanceAfter)
	return nil
}

// SubtractBalance mengurangi saldo pengguna.
// Akan mengembalikan ErrInsufficientFunds jika saldo tidak cukup.
//
// txType mengidentifikasi alasan pengurangan (misal: "poker_buyin").
// reference memberikan konteks tambahan.
func (s *EconomyService) SubtractBalance(jid string, amount int, txType, reference string) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi subtract: %w", err)
	}
	defer tx.Rollback()

	// Gunakan UPDATE ... WHERE balance >= amount untuk atomik check-and-subtract
	res, err := tx.Exec("UPDATE users SET balance = balance - ? WHERE jid = ? AND balance >= ?", amount, jid, amount)
	if err != nil {
		return fmt.Errorf("gagal mengurangi saldo: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal cek hasil update: %w", err)
	}

	if rows == 0 {
		// Pastikan user terdaftar, jika terdaftar berarti saldo tak cukup
		var current int
		err := tx.QueryRow("SELECT balance FROM users WHERE jid = ?", jid).Scan(&current)
		if err == sql.ErrNoRows {
			// Daftarkan dulu — unlock mutex karena registerNewUser perlu lock
			tx.Rollback()
			s.mu.Unlock()
			_, errReg := s.GetBalance(jid)
			s.mu.Lock()
			if errReg != nil {
				return errReg
			}
			return ErrInsufficientFunds
		}
		return ErrInsufficientFunds
	}

	// Baca saldo setelah update
	var balanceAfter int
	if err := tx.QueryRow("SELECT balance FROM users WHERE jid = ?", jid).Scan(&balanceAfter); err != nil {
		return fmt.Errorf("gagal baca saldo setelah subtract: %w", err)
	}

	if err := logTransaction(tx, jid, -amount, balanceAfter, txType, reference); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit subtract balance: %w", err)
	}

	log.Printf("[ECONOMY] %s -%d (%s: %s) → saldo: %d", jid, amount, txType, reference, balanceAfter)
	return nil
}

// Transfer memindahkan saldo dari pengirim ke penerima.
// Kedua sisi transaksi dicatat secara atomik dalam satu database transaction.
func (s *EconomyService) Transfer(fromJid, toJid string, amount int) error {
	if fromJid == toJid {
		return ErrSelfTransfer
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Pastikan keduanya terdaftar
	if _, err := s.GetBalance(fromJid); err != nil {
		return err
	}
	if _, err := s.GetBalance(toJid); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi transfer: %w", err)
	}
	defer tx.Rollback()

	// Kurangi dari pengirim
	res, err := tx.Exec("UPDATE users SET balance = balance - ? WHERE jid = ? AND balance >= ?", amount, fromJid, amount)
	if err != nil {
		return fmt.Errorf("gagal memotong saldo pengirim: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrInsufficientFunds
	}

	// Tambah ke penerima
	_, err = tx.Exec("UPDATE users SET balance = balance + ? WHERE jid = ?", amount, toJid)
	if err != nil {
		return fmt.Errorf("gagal menambah saldo penerima: %w", err)
	}

	// Baca saldo setelah update untuk kedua pihak
	var fromBalance, toBalance int
	if err := tx.QueryRow("SELECT balance FROM users WHERE jid = ?", fromJid).Scan(&fromBalance); err != nil {
		return fmt.Errorf("gagal baca saldo pengirim: %w", err)
	}
	if err := tx.QueryRow("SELECT balance FROM users WHERE jid = ?", toJid).Scan(&toBalance); err != nil {
		return fmt.Errorf("gagal baca saldo penerima: %w", err)
	}

	// Log kedua sisi transaksi
	ref := fmt.Sprintf("transfer_%s_to_%s", fromJid, toJid)
	if err := logTransaction(tx, fromJid, -amount, fromBalance, "transfer_out", ref); err != nil {
		return err
	}
	if err := logTransaction(tx, toJid, amount, toBalance, "transfer_in", ref); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transfer: %w", err)
	}

	log.Printf("[ECONOMY] Transfer %d: %s → %s (saldo: %d → %d)", amount, fromJid, toJid, fromBalance, toBalance)
	return nil
}

// GetTransactionHistory mengembalikan riwayat transaksi terakhir untuk seorang user.
// Berguna untuk debugging dan fitur "riwayat" di masa depan.
func (s *EconomyService) GetTransactionHistory(jid string, limit int) ([]TransactionLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		"SELECT id, jid, amount, balance_after, type, reference, created_at FROM transaction_logs WHERE jid = ? ORDER BY id DESC LIMIT ?",
		jid, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat transaksi: %w", err)
	}
	defer rows.Close()

	var logs []TransactionLog
	for rows.Next() {
		var l TransactionLog
		if err := rows.Scan(&l.ID, &l.JID, &l.Amount, &l.BalanceAfter, &l.Type, &l.Reference, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// logTransaction mencatat satu mutasi saldo ke tabel transaction_logs.
// Selalu dipanggil di dalam database transaction yang sama dengan mutasi saldo,
// sehingga log dan saldo dijamin konsisten (keduanya commit atau rollback bersama).
func logTransaction(tx *sql.Tx, jid string, amount, balanceAfter int, txType, reference string) error {
	_, err := tx.Exec(
		"INSERT INTO transaction_logs (jid, amount, balance_after, type, reference) VALUES (?, ?, ?, ?, ?)",
		jid, amount, balanceAfter, txType, reference,
	)
	if err != nil {
		return fmt.Errorf("gagal mencatat transaksi: %w", err)
	}
	return nil
}

// SetBalance langsung menetapkan saldo pengguna (untuk admin panel).
func (s *EconomyService) SetBalance(jid string, amount int, reference string) error {
	if amount < 0 {
		return ErrInvalidAmount
	}

	// Pastikan user terdaftar
	_, err := s.GetBalance(jid)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi set balance: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE users SET balance = ? WHERE jid = ?", amount, jid)
	if err != nil {
		return fmt.Errorf("gagal set balance: %w", err)
	}

	if err := logTransaction(tx, jid, amount, amount, "admin_set", reference); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit set balance: %w", err)
	}

	log.Printf("[ECONOMY] Admin set balance %s to %d (%s)", jid, amount, reference)
	return nil
}

// SearchUsers mencari pengguna berdasarkan nama atau JID (untuk admin panel).
func (s *EconomyService) SearchUsers(query string) ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows *sql.Rows
	var err error
	if query == "" {
		rows, err = s.db.Query("SELECT jid, name, balance FROM users ORDER BY balance DESC")
	} else {
		likeQuery := "%" + query + "%"
		rows, err = s.db.Query("SELECT jid, name, balance FROM users WHERE name LIKE ? OR jid LIKE ? ORDER BY balance DESC", likeQuery, likeQuery)
	}

	if err != nil {
		return nil, fmt.Errorf("gagal mencari user: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.JID, &u.Name, &u.Balance); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// DB mengembalikan instance sql.DB untuk diakses oleh servis lain (seperti MiningService).
func (s *EconomyService) DB() *sql.DB {
	return s.db
}
