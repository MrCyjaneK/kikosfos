package wallet

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// paymentSeen asks the coin whether address holds amount atoms.
// Tests replace it so a claim check does not use the chain.
var paymentSeen = func(c Coin, address, amount string) (bool, error) {
	return c.Received(address, amount)
}

// A claimed ride is the one this driver accepted. The fare is checked
// against the ride address, not the driver's own wallet.
type heldRide struct {
	Server string
	ID     string
	Coin   int
	PayTo  string
	Fare   uint64
	Paid   bool
}

var (
	heldMu sync.Mutex
	held   = map[string]heldRide{}
)

func clearHeld() {
	heldMu.Lock()
	held = map[string]heldRide{}
	heldMu.Unlock()
}

func rideKey(server, id string) string {
	return strings.ToLower(strings.TrimSpace(server)) + "\n" + strings.TrimSpace(id)
}

func readyFor(host string) (readyServer, error) {
	host = strings.TrimSpace(host)
	for _, server := range readyCopy() {
		if strings.EqualFold(server.Host, host) {
			return server, nil
		}
	}
	return readyServer{}, errors.New("unknown server")
}

func rememberHeld(ride heldRide) {
	heldMu.Lock()
	held[rideKey(ride.Server, ride.ID)] = ride
	heldMu.Unlock()
}

func heldFor(server, id string) (heldRide, bool) {
	heldMu.Lock()
	defer heldMu.Unlock()
	ride, ok := held[rideKey(server, id)]
	return ride, ok
}

func markHeldPaid(server, id string) {
	heldMu.Lock()
	defer heldMu.Unlock()
	key := rideKey(server, id)
	ride, ok := held[key]
	if !ok {
		return
	}
	ride.Paid = true
	held[key] = ride
}

// etaMinutes is the pickup time at 30 km/h, and at least one minute.
func etaMinutes(km float64) int {
	minutes := int(math.Ceil(km * 2))
	if minutes < 1 {
		return 1
	}
	return minutes
}

// ClaimDrive accepts the ride on its server.
// pickupKm is how far the driver is from the pickup, sent as the ETA.
func ClaimDrive(server, id, pickupKm string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("unknown ride")
	}
	ready, err := readyFor(server)
	if err != nil {
		return err
	}
	km, err := strconv.ParseFloat(strings.TrimSpace(pickupKm), 64)
	if err != nil || km < 0 || math.IsNaN(km) || math.IsInf(km, 0) {
		km = 0
	}
	body, err := json.Marshal(struct {
		EtaMinutes int     `json:"eta_minutes"`
		EtaKm      float64 `json:"eta_km"`
	}{etaMinutes(km), km})
	if err != nil {
		return err
	}
	status, resp, err := kikoDo(http.MethodPost, ready.Host, "/api/v1/driver/drive/"+url.PathEscape(id), body, map[string]string{
		"X-Driver-Key": ready.Key,
	})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return errors.New(ready.Host + ": " + apiError(status, resp))
	}
	var view struct {
		PayTo string `json:"pay_to"`
		Fare  uint64 `json:"fare_lovelace"`
	}
	if json.Unmarshal(resp, &view) != nil || view.PayTo == "" || view.Fare == 0 {
		return errors.New(ready.Host + ": bad ride")
	}
	rememberHeld(heldRide{
		Server: ready.Host,
		ID:     id,
		Coin:   ready.Coin,
		PayTo:  view.PayTo,
		Fare:   view.Fare,
	})
	return nil
}

// VerifyDrive is true when the ride address holds the fare.
// The ride must already be claimed. A paid result is remembered.
func VerifyDrive(server, id string) (bool, error) {
	ride, ok := heldFor(server, id)
	if !ok {
		return false, errors.New("claim the ride first")
	}
	if ride.Paid {
		return true, nil
	}
	c, err := coinAt(ride.Coin)
	if err != nil {
		return false, err
	}
	paid, err := paymentSeen(c, ride.PayTo, strconv.FormatUint(ride.Fare, 10))
	if err != nil || !paid {
		return false, err
	}
	markHeldPaid(server, id)
	return true, nil
}

// DriveClaimed is 1 when this driver has accepted the ride.
func DriveClaimed(server, id string) int {
	if _, ok := heldFor(server, id); ok {
		return 1
	}
	return 0
}

// DrivePaid is 1 when the fare was seen on the ride address.
func DrivePaid(server, id string) int {
	ride, ok := heldFor(server, id)
	if ok && ride.Paid {
		return 1
	}
	return 0
}
