package wallet

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Car is the driver profile stored with the servers and fees.
type Car struct {
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Color        string `json:"color"`
	LicensePlate string `json:"license_plate_number"`
	Year         int    `json:"year"`
}

type serverRecord struct {
	Host     string `json:"host"`
	Currency string `json:"currency"`
	BaseFee  string `json:"base_fee,omitempty"`
	FeePerKm string `json:"fee_per_km,omitempty"`
}

type profileFile struct {
	Car     Car            `json:"car"`
	Servers []serverRecord `json:"servers"`
}

func emptyProfile() profileFile {
	return profileFile{Servers: []serverRecord{}}
}

func normalizeDoc(doc *profileFile) {
	if doc.Servers == nil {
		doc.Servers = []serverRecord{}
	}
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
	if err != nil || year < 0 || year > 9999 {
		return 0, errors.New("enter a year")
	}
	return year, nil
}
