package wallet

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
	_ "modernc.org/sqlite"
)

// Store is the on-disk wallet files for one process.
// ConfigDir holds the PIN verifier and encrypted seeds.
// DataDir holds one database per coin.
type Store struct {
	ConfigDir string
	DataDir   string
}

func (s *Store) ensureDirs() error {
	for _, dir := range []string{
		s.ConfigDir,
		filepath.Join(s.ConfigDir, "seeds"),
		s.DataDir,
		filepath.Join(s.DataDir, "wallets"),
		filepath.Join(s.DataDir, "locks"),
	} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) pinPath() string { return filepath.Join(s.ConfigDir, "pin") }

func (s *Store) pinLockPath() string { return filepath.Join(s.ConfigDir, "pin.lock") }

func (s *Store) seedPath(id string) string {
	return filepath.Join(s.ConfigDir, "seeds", id)
}

func (s *Store) dbPath(id string) string {
	return filepath.Join(s.DataDir, "wallets", id+".db")
}

func (s *Store) coinLockPath(id string) string {
	return filepath.Join(s.DataDir, "locks", id+".lock")
}

func lockFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func unlockFile(f *os.File) {
	if f == nil {
		return
	}
	_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
	_ = f.Close()
}

func (s *Store) withCoin(id string, fn func() error) error {
	f, err := lockFile(s.coinLockPath(id))
	if err != nil {
		return err
	}
	defer unlockFile(f)
	return fn()
}

func (s *Store) PinIsSet() (bool, error) {
	f, err := lockFile(s.pinLockPath())
	if err != nil {
		return false, err
	}
	defer unlockFile(f)
	_, err = os.Stat(s.pinPath())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetupPIN derives a key, then writes the verifier if one does not exist yet.
// The KDF runs outside the lock.
func (s *Store) SetupPIN(pin string) ([]byte, error) {
	salt, err := randomBytes(16)
	if err != nil {
		return nil, err
	}
	params := activeKDF
	key := deriveKey(pin, salt, params)
	blob, err := encodePIN(params, salt, key)
	if err != nil {
		return nil, err
	}
	f, err := lockFile(s.pinLockPath())
	if err != nil {
		return nil, err
	}
	defer unlockFile(f)
	if _, err := os.Stat(s.pinPath()); err == nil {
		return nil, errors.New("pin is already set")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := writeFileAtomic(s.pinPath(), blob); err != nil {
		return nil, err
	}
	return key, nil
}

// UnlockPIN reads the verifier, releases the lock, then runs the KDF.
func (s *Store) UnlockPIN(pin string) ([]byte, error) {
	f, err := lockFile(s.pinLockPath())
	if err != nil {
		return nil, err
	}
	blob, readErr := os.ReadFile(s.pinPath())
	unlockFile(f)
	if errors.Is(readErr, os.ErrNotExist) {
		return nil, errors.New("pin is not set")
	}
	if readErr != nil {
		return nil, readErr
	}
	params, salt, nonce, ciphertext, err := decodePIN(blob)
	if err != nil {
		return nil, err
	}
	key := deriveKey(pin, salt, params)
	plain, err := open(key, nonce, ciphertext)
	if err != nil || string(plain) != canary {
		return nil, errors.New("wrong pin")
	}
	return key, nil
}

func (s *Store) seedExists(id string) bool {
	_, err := os.Stat(s.seedPath(id))
	return err == nil
}

func (s *Store) Exists(id string) (bool, error) {
	var ok bool
	err := s.withCoin(id, func() error {
		ok = s.seedExists(id)
		return nil
	})
	return ok, err
}

func (s *Store) writeSeed(id string, key []byte, mnemonic string) error {
	blob, err := encodeSeed(key, mnemonic)
	if err != nil {
		return err
	}
	return writeFileAtomic(s.seedPath(id), blob)
}

func (s *Store) readSeed(id string, key []byte) (string, error) {
	blob, err := os.ReadFile(s.seedPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("wallet does not exist")
	}
	if err != nil {
		return "", err
	}
	return decodeSeed(key, blob)
}

var errNoPublic = errors.New("wallet record is missing")

func (s *Store) writePublic(id string, pub Public) error {
	body, err := json.Marshal(publicRecord{Address: pub.Address, Details: pub.Details})
	if err != nil {
		return err
	}
	db, err := s.openDB(id)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO wallet (id, body) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET body = excluded.body`, string(body))
	return err
}

func (s *Store) readPublic(id string) (Public, error) {
	if _, err := os.Stat(s.dbPath(id)); errors.Is(err, os.ErrNotExist) {
		return Public{}, errNoPublic
	}
	db, err := s.openDB(id)
	if err != nil {
		return Public{}, err
	}
	defer db.Close()
	var body string
	err = db.QueryRow(`SELECT body FROM wallet WHERE id = 1`).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return Public{}, errNoPublic
	}
	if err != nil {
		return Public{}, err
	}
	var rec publicRecord
	if err := json.Unmarshal([]byte(body), &rec); err != nil {
		return Public{}, err
	}
	return Public{Address: rec.Address, Details: rec.Details}, nil
}

type publicRecord struct {
	Address string   `json:"address"`
	Details []Detail `json:"details"`
}

// SaveWallet writes the seed and the public record while holding the coin lock.
func (s *Store) SaveWallet(id string, key []byte, mnemonic string, pub Public) error {
	return s.withCoin(id, func() error {
		if s.seedExists(id) {
			return errors.New("wallet already exists")
		}
		if err := s.writeSeed(id, key, mnemonic); err != nil {
			return err
		}
		if err := s.writePublic(id, pub); err != nil {
			_ = os.Remove(s.seedPath(id))
			return err
		}
		return nil
	})
}

// Public returns the stored address. The bool is false when no seed exists.
// A missing database is rebuilt from the seed when key is set.
func (s *Store) Public(id string, key []byte, restore func(mnemonic string) (Public, error)) (Public, bool, error) {
	var pub Public
	var exists bool
	err := s.withCoin(id, func() error {
		if !s.seedExists(id) {
			return nil
		}
		exists = true
		got, err := s.readPublic(id)
		if err == nil {
			pub = got
			return nil
		}
		if !errors.Is(err, errNoPublic) {
			return err
		}
		if len(key) == 0 || restore == nil {
			return errNoPublic
		}
		mnemonic, err := s.readSeed(id, key)
		if err != nil {
			return err
		}
		rebuilt, err := restore(mnemonic)
		if err != nil {
			return err
		}
		if err := s.writePublic(id, rebuilt); err != nil {
			return err
		}
		pub = rebuilt
		return nil
	})
	return pub, exists, err
}

func (s *Store) Mnemonic(id string, key []byte) (string, error) {
	var mnemonic string
	err := s.withCoin(id, func() error {
		got, err := s.readSeed(id, key)
		mnemonic = got
		return err
	})
	return mnemonic, err
}

func (s *Store) Delete(id string) error {
	return s.withCoin(id, func() error {
		if err := os.Remove(s.seedPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Remove(s.dbPath(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
}

func (s *Store) openDB(id string) (*sql.DB, error) {
	path := s.dbPath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS wallet (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		body TEXT NOT NULL
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func sqliteDSN(path string) string {
	u := url.URL{Scheme: "file", Path: path, RawQuery: "_pragma=busy_timeout(5000)"}
	return u.String()
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(name)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0600); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	cleanup = false
	return nil
}
