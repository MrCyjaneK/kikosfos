package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Car is the driver profile stored with the servers and fees.
type Car struct {
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Color        string `json:"color"`
	LicensePlate string `json:"license_plate_number"`
	Year         int    `json:"year"`
}

// chainFees is one chain's prices on a server. Empty fees mean the driver
// does not take rides on that chain.
type chainFees struct {
	Currency string `json:"currency"`
	BaseFee  string `json:"base_fee,omitempty"`
	FeePerKm string `json:"fee_per_km,omitempty"`
}

type serverRecord struct {
	Host   string      `json:"host"`
	Chains []chainFees `json:"chains,omitempty"`
	// Currency, BaseFee and FeePerKm are the shape written before a server
	// could offer more than one chain. They are read once and stored as chains.
	Currency string `json:"currency,omitempty"`
	BaseFee  string `json:"base_fee,omitempty"`
	FeePerKm string `json:"fee_per_km,omitempty"`
}

// savedRide is a ride this driver has accepted. The offer itself is loaded
// from the server, so a restart still shows it after it leaves the open list.
type savedRide struct {
	Server string `json:"server"`
	ID     string `json:"id"`
}

type profileFile struct {
	Car     Car            `json:"car"`
	Servers []serverRecord `json:"servers"`
	GPS     string         `json:"gps,omitempty"`
	Rides   []savedRide    `json:"rides,omitempty"`
}

func emptyProfile() profileFile {
	return profileFile{Servers: []serverRecord{}}
}

func normalizeDoc(doc *profileFile) {
	if doc.Servers == nil {
		doc.Servers = []serverRecord{}
	}
	for i := range doc.Servers {
		doc.Servers[i].migrate()
	}
}

func (s *serverRecord) migrate() {
	if len(s.Chains) == 0 && s.Currency != "" {
		s.Chains = []chainFees{{
			Currency: s.Currency,
			BaseFee:  s.BaseFee,
			FeePerKm: s.FeePerKm,
		}}
	}
	if s.Chains == nil {
		s.Chains = []chainFees{}
	}
	s.Chains = uniqueChains(s.Chains)
	s.Currency = ""
	s.BaseFee = ""
	s.FeePerKm = ""
}

// uniqueChains keeps the first fees for each coin. A chain stored twice
// would ask the server twice and show the same ride twice.
func uniqueChains(in []chainFees) []chainFees {
	out := make([]chainFees, 0, len(in))
	seen := map[string]bool{}
	for _, chain := range in {
		name := strings.ToLower(strings.TrimSpace(chain.Currency))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		chain.Currency = name
		out = append(out, chain)
	}
	return out
}

func (s serverRecord) chain(currency string) (chainFees, bool) {
	for _, chain := range s.Chains {
		if chain.Currency == currency {
			return chain, true
		}
	}
	return chainFees{}, false
}

func (s *Store) profilePath() string {
	return filepath.Join(s.ConfigDir, "profile")
}

func (s *Store) profileLockPath() string {
	return filepath.Join(s.ConfigDir, "profile.lock")
}

func (s *Store) withProfile(fn func() error) error {
	f, err := lockFile(s.profileLockPath())
	if err != nil {
		return err
	}
	defer unlockFile(f)
	return fn()
}

func (s *Store) readProfile(key []byte) (profileFile, error) {
	blob, err := os.ReadFile(s.profilePath())
	if errors.Is(err, os.ErrNotExist) {
		return emptyProfile(), nil
	}
	if err != nil {
		return profileFile{}, err
	}
	plain, err := decodeProfile(key, blob)
	if err != nil {
		return profileFile{}, err
	}
	var doc profileFile
	if err := json.Unmarshal(plain, &doc); err != nil {
		return profileFile{}, errors.New("profile is corrupt")
	}
	normalizeDoc(&doc)
	return doc, nil
}

func (s *Store) writeProfile(key []byte, doc profileFile) error {
	normalizeDoc(&doc)
	plain, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	blob, err := encodeProfile(key, plain)
	if err != nil {
		return err
	}
	return writeFileAtomic(s.profilePath(), blob)
}

func loadProfile() (profileFile, error) {
	key, err := sessionKey()
	if err != nil {
		return profileFile{}, err
	}
	s, err := openStore()
	if err != nil {
		return profileFile{}, err
	}
	var doc profileFile
	err = s.withProfile(func() error {
		var readErr error
		doc, readErr = s.readProfile(key)
		return readErr
	})
	return doc, err
}

func updateProfile(fn func(*profileFile) error) error {
	key, err := sessionKey()
	if err != nil {
		return err
	}
	s, err := openStore()
	if err != nil {
		return err
	}
	return s.withProfile(func() error {
		doc, err := s.readProfile(key)
		if err != nil {
			return err
		}
		if err := fn(&doc); err != nil {
			return err
		}
		return s.writeProfile(key, doc)
	})
}

func ProfileCar() (Car, error) {
	doc, err := loadProfile()
	if err != nil {
		return Car{}, err
	}
	return doc.Car, nil
}

func ProfileBrand() (string, error) {
	car, err := ProfileCar()
	if err != nil {
		return "", err
	}
	return car.Brand, nil
}

func ProfileModel() (string, error) {
	car, err := ProfileCar()
	if err != nil {
		return "", err
	}
	return car.Model, nil
}

func ProfileColor() (string, error) {
	car, err := ProfileCar()
	if err != nil {
		return "", err
	}
	return car.Color, nil
}

func ProfilePlate() (string, error) {
	car, err := ProfileCar()
	if err != nil {
		return "", err
	}
	return car.LicensePlate, nil
}

// ProfileYear is empty when the year has not been set.
func ProfileYear() (string, error) {
	car, err := ProfileCar()
	if err != nil {
		return "", err
	}
	if car.Year == 0 {
		return "", nil
	}
	return strconv.Itoa(car.Year), nil
}

// SaveCar stores the car and leaves fees and servers as they are.
func SaveCar(brand, model, color, plate, yearText string) error {
	year, err := parseYear(yearText)
	if err != nil {
		return err
	}
	car := Car{
		Brand:        strings.TrimSpace(brand),
		Model:        strings.TrimSpace(model),
		Color:        strings.TrimSpace(color),
		LicensePlate: strings.TrimSpace(plate),
		Year:         year,
	}
	return updateProfile(func(doc *profileFile) error {
		doc.Car = car
		return nil
	})
}

func parseYear(text string) (int, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	year, err := strconv.Atoi(text)
	maxYear := time.Now().Year() + 1
	if err != nil || year < 1900 || year > maxYear {
		return 0, fmt.Errorf("enter a year from 1900 to %d", maxYear)
	}
	return year, nil
}
