package wallet

import (
	"encoding/json"
	"errors"
	"net/http"
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

// DriverRequest is the JSON body for POST /api/v1/driver on the server's first chain.
func DriverRequest(host string) ([]byte, error) {
	return driverRequest(host, "")
}

func driverRequest(host, currency string) ([]byte, error) {
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
	chain := firstChain(server)
	if currency != "" {
		var ok bool
		chain, ok = server.chain(currency)
		if !ok {
			return nil, errors.New("unknown server")
		}
	}
	index, _, err := walletForCurrency(chain.Currency)
	if err != nil {
		return nil, err
	}
	if !carComplete(doc.Car) {
		return nil, errors.New("car is incomplete")
	}
	if chain.BaseFee == "" || chain.FeePerKm == "" {
		return nil, errors.New("fees are not set")
	}
	base, err := strconv.ParseUint(chain.BaseFee, 10, 64)
	if err != nil {
		return nil, errors.New("fees are not set")
	}
	perKm, err := strconv.ParseUint(chain.FeePerKm, 10, 64)
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

// RegisterDriver posts the server's first chain to POST /api/v1/driver.
// Posting again replaces that driver's fees and car. The signature is the
// account key in the body, so the request carries no X-Driver-Key.
func RegisterDriver(host string) ([]byte, error) {
	return registerChain(host, "")
}

func registerChain(host, currency string) ([]byte, error) {
	body, err := driverRequest(host, currency)
	if err != nil {
		return nil, err
	}
	host, err = normalizeHost(host)
	if err != nil {
		return nil, err
	}
	chainCurrency := currency
	if chainCurrency == "" {
		doc, err := loadProfile()
		if err != nil {
			return nil, err
		}
		for _, item := range doc.Servers {
			if strings.EqualFold(item.Host, host) {
				chainCurrency = firstChain(item).Currency
				break
			}
		}
	}
	coin, _, err := walletForCurrency(chainCurrency)
	if err != nil {
		return nil, err
	}
	status, resp, err := driverDo(http.MethodPost, host, "/api/v1/driver", body, coin, false)
	if err != nil {
		return nil, err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return errorObject(status, resp), nil
	}
	if len(resp) == 0 {
		return []byte("{}"), nil
	}
	return resp, nil
}
