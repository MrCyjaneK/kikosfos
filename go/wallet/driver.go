package wallet

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type driverBody struct {
	PublicKey string    `json:"public_key"`
	BaseFee   uint64    `json:"base_fee"`
	FeePerKm  uint64    `json:"fee_per_km"`
	Car       driverCar `json:"car"`
}

type driverCar struct {
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Color        string `json:"color"`
	LicensePlate string `json:"license_plate_number"`
	Year         int    `json:"year"`
}

// postDriver pretends POST /api/v1/driver. The body is not sent.
// Tests replace it with a server error object.
var postDriver = func([]byte) ([]byte, error) {
	return []byte("{}"), nil
}

// DriverRequest is the JSON body for POST /api/v1/driver.
func DriverRequest(host string) ([]byte, error) {
	host, err := normalizeHost(host)
	if err != nil {
		return nil, err
	}
	doc, err := loadProfile()
	if err != nil {
		return nil, err
	}
	var server serverRecord
	found := false
	for _, item := range doc.Servers {
		if strings.EqualFold(item.Host, host) {
			server = item
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("unknown server")
	}
	index, _, err := walletForCurrency(server.Currency)
	if err != nil {
		return nil, err
	}
	if !carComplete(doc.Car) {
		return nil, errors.New("car is incomplete")
	}
	if server.BaseFee == "" || server.FeePerKm == "" {
		return nil, errors.New("fees are not set")
	}
	base, err := strconv.ParseUint(server.BaseFee, 10, 64)
	if err != nil {
		return nil, errors.New("fees are not set")
	}
	perKm, err := strconv.ParseUint(server.FeePerKm, 10, 64)
	if err != nil {
		return nil, errors.New("fees are not set")
	}
	publicKey, err := WalletMultisigMaterial(index)
	if err != nil {
		return nil, err
	}
	body := driverBody{
		PublicKey: publicKey,
		BaseFee:   base,
		FeePerKm:  perKm,
		Car: driverCar{
			Brand:        doc.Car.Brand,
			Model:        doc.Car.Model,
			Color:        doc.Car.Color,
			LicensePlate: doc.Car.LicensePlate,
			Year:         doc.Car.Year,
		},
	}
	return json.Marshal(body)
}

func carComplete(car Car) bool {
	return strings.TrimSpace(car.Brand) != "" &&
		strings.TrimSpace(car.Model) != "" &&
		strings.TrimSpace(car.Color) != "" &&
		strings.TrimSpace(car.LicensePlate) != "" &&
		car.Year > 0
}

// RegisterDriver builds the driver body and pretends to post it.
// The mock response is an empty JSON object.
func RegisterDriver(host string) ([]byte, error) {
	body, err := DriverRequest(host)
	if err != nil {
		return nil, err
	}
	return postDriver(body)
}
