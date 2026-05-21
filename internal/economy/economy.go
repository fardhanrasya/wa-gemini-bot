package economy

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const (
	InitialBalance = 5000 // Modal awal yang diberikan ke pemain baru
)

var (
	ErrInsufficientFunds = errors.New("saldo tidak mencukupi")
	ErrSelfTransfer      = errors.New("tidak bisa transfer ke diri sendiri")
	ErrInvalidAmount     = errors.New("jumlah tidak valid")
)

// EconomyService mengelola mata uang/chip pemain secara persisten
// menggunakan database SQLite.
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

// NewEconomyService membuat atau membuka koneksi ke database SQLite.
func NewEconomyService(dbPath string) (*EconomyService, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database economy: %w", err)
	}

	// Buat tabel jika belum ada
	query := `
	CREATE TABLE IF NOT EXISTS users (
		jid TEXT PRIMARY KEY,
		name TEXT DEFAULT '',
		balance INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("gagal membuat tabel users: %w", err)
	}

	return &EconomyService{
		db: db,
	}, nil
}

// Close menutup koneksi database.
func (s *EconomyService) Close() error {
	return s.db.Close()
}

// UpdateName mengupdate nama tampilan pengguna untuk keperluan leaderboard.
func (s *EconomyService) UpdateName(jid, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec("UPDATE users SET name = ? WHERE jid = ?", name, jid)
	return err
}

// GetLeaderboard mengembalikan daftar pemain dengan saldo tertinggi.
func (s *EconomyService) GetLeaderboard(limit int) ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query("SELECT jid, name, balance FROM users ORDER BY balance DESC LIMIT ?", limit)
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

	_, err = s.db.Exec("INSERT INTO users (jid, balance) VALUES (?, ?)", jid, InitialBalance)
	if err != nil {
		return fmt.Errorf("gagal register user baru: %w", err)
	}
	log.Printf("[ECONOMY] User baru %s mendapatkan modal awal %d", jid, InitialBalance)
	return nil
}

// AddBalance menambahkan sejumlah saldo ke pengguna.
func (s *EconomyService) AddBalance(jid string, amount int) error {
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

	_, err = s.db.Exec("UPDATE users SET balance = balance + ? WHERE jid = ?", amount, jid)
	if err != nil {
		return fmt.Errorf("gagal menambah saldo: %w", err)
	}
	return nil
}

// SubtractBalance mengurangi saldo pengguna.
// Akan mengembalikan ErrInsufficientFunds jika saldo tidak cukup.
func (s *EconomyService) SubtractBalance(jid string, amount int) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Kita bisa gunakan transaksi untuk membaca lalu update, atau
	// andalkan RDBMS dengan WHERE balance >= amount.
	res, err := s.db.Exec("UPDATE users SET balance = balance - ? WHERE jid = ? AND balance >= ?", amount, jid, amount)
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
		err := s.db.QueryRow("SELECT balance FROM users WHERE jid = ?", jid).Scan(&current)
		if err == sql.ErrNoRows {
			// Daftarkan dulu
			s.mu.Unlock() // unlock to allow register
			_, errReg := s.GetBalance(jid)
			s.mu.Lock()   // re-lock
			if errReg != nil {
				return errReg
			}
			return ErrInsufficientFunds
		}
		return ErrInsufficientFunds
	}

	return nil
}

// Transfer memindahkan saldo dari pengirim ke penerima.
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
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInsufficientFunds
	}

	// Tambah ke penerima
	_, err = tx.Exec("UPDATE users SET balance = balance + ? WHERE jid = ?", amount, toJid)
	if err != nil {
		return fmt.Errorf("gagal menambah saldo penerima: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transfer: %w", err)
	}

	return nil
}
