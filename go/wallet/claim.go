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
	"time"

	"kiko-sfos/wallet/cardano"
)

// paymentSeen asks the coin whether address holds amount atoms.
// Tests replace it so a claim check does not use the chain.
var paymentSeen = func(c Coin, address, amount string) (bool, error) {
	return c.Received(address, amount)
}

// A claimed ride is the one this driver accepted. The fare is checked
// against the ride address, not the driver's own wallet. Offer is what
// the home screen shows after the ride leaves the available list.
type heldRide struct {
	Offer  driveOffer
	PayTo  string
	Script string
	Index  uint32
	Ready  bool
	Paid   bool
	Status string
}

// errRideGone means the saved ride is finished or no longer on the server.
var errRideGone = errors.New("ride is gone")

// buildPayout signs a spend of the ride address. Tests replace it.
var buildPayout = cardano.Payout

var (
	heldMu    sync.Mutex
	held      = map[string]heldRide{}
	heldOrder []string
)

func clearHeld() {
	heldMu.Lock()
	held = map[string]heldRide{}
	heldOrder = nil
	heldMu.Unlock()
}

func rideKey(server, id string) string {
	return strings.ToLower(strings.TrimSpace(server)) + "\n" + strings.TrimSpace(id)
}

func dropOffer(server, id string) {
	driveMu.Lock()
	defer driveMu.Unlock()
	next := make([]driveOffer, 0, len(driveOffers))
	for _, row := range driveOffers {
		if strings.EqualFold(row.Server, server) && row.ID == id {
			continue
		}
		next = append(next, row)
	}
	driveOffers = next
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
	defer heldMu.Unlock()
	key := rideKey(ride.Offer.Server, ride.Offer.ID)
	if _, ok := held[key]; !ok {
		heldOrder = append(heldOrder, key)
	}
	held[key] = ride
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
	ride.Status = "paid"
	held[key] = ride
}

func dropHeld(server, id string) {
	heldMu.Lock()
	defer heldMu.Unlock()
	key := rideKey(server, id)
	if _, ok := held[key]; !ok {
		return
	}
	delete(held, key)
	next := make([]string, 0, len(heldOrder))
	for _, item := range heldOrder {
		if item != key {
			next = append(next, item)
		}
	}
	heldOrder = next
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
	view, err := parseRideView(resp)
	if err != nil || view.PayTo == "" || view.Fare == 0 {
		return errors.New(ready.Host + ": bad ride")
	}
	dropOffer(ready.Host, id)
	rememberHeld(heldFromView(ready, id, km, "accepted", view))
	if err := saveActiveRide(ready.Host, id); err != nil {
		return err
	}
	return nil
}

type rideView struct {
	Status       string `json:"status"`
	StartAddress string `json:"start_address"`
	EndAddress   string `json:"end_address"`
	Start        struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"start"`
	End struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"end"`
	DistanceKm   float64  `json:"distance_km"`
	PickupTime   string   `json:"pickup_time"`
	PayTo        string   `json:"pay_to"`
	Fare         uint64   `json:"fare_lovelace"`
	EtaKm        *float64 `json:"eta_km"`
	AddressIndex *uint32  `json:"address_index"`
	Script       string   `json:"script"`
}

func parseRideView(body []byte) (rideView, error) {
	var view rideView
	if err := json.Unmarshal(body, &view); err != nil {
		return rideView{}, err
	}
	return view, nil
}

func heldFromView(ready readyServer, id string, pickupKm float64, status string, view rideView) heldRide {
	when, err := time.Parse(time.RFC3339, view.PickupTime)
	if err != nil {
		when, err = time.Parse(time.RFC3339Nano, view.PickupTime)
	}
	if err != nil {
		when = time.Now()
	}
	ride := heldRide{
		Offer: driveOffer{
			ID:           id,
			Server:       ready.Host,
			Coin:         ready.Coin,
			StartAddress: view.StartAddress,
			EndAddress:   view.EndAddress,
			PickupKm:     pickupKm,
			TripM:        kmToMeters(view.DistanceKm),
			When:         when,
			StartLat:     view.Start.Lat,
			StartLon:     view.Start.Lng,
			EndLat:       view.End.Lat,
			EndLon:       view.End.Lng,
			Immediate:    !when.After(time.Now().Add(nowWindow)),
			Fare:         view.Fare,
			Base:         ready.Base,
			PerKm:        ready.PerKm,
		},
		PayTo:  view.PayTo,
		Script: view.Script,
		Status: status,
		Paid:   status == "paid" || status == "paying_out",
	}
	if view.AddressIndex != nil {
		ride.Index = *view.AddressIndex
		ride.Ready = true
	}
	return ride
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
	c, err := coinAt(ride.Offer.Coin)
	if err != nil {
		return false, err
	}
	paid, err := paymentSeen(c, ride.PayTo, strconv.FormatUint(ride.Offer.Fare, 10))
	if err != nil || !paid {
		return false, err
	}
	markHeldPaid(server, id)
	return true, nil
}

// CashOutDrive spends the ride address and asks the server to cosign.
// The ride must already be paid. On success it leaves the active list.
func CashOutDrive(server, id string) error {
	ride, ok := heldFor(server, id)
	if !ok {
		return errors.New("claim the ride first")
	}
	if !ride.Paid {
		return errors.New("the ride is not paid")
	}
	if !ride.Ready || ride.Script == "" {
		return errors.New("this ride has no payout script")
	}
	c, err := coinAt(ride.Offer.Coin)
	if err != nil {
		return err
	}
	if c.ID() != cardano.ID {
		return errors.New("cash out is only available for cardano")
	}
	mnemonic, err := WalletMnemonic(ride.Offer.Coin)
	if err != nil {
		return err
	}
	signed, err := buildPayout(mnemonic, ride.PayTo, ride.Script, ride.Index)
	if err != nil {
		return err
	}
	ready, err := readyFor(server)
	if err != nil {
		return err
	}
	body, err := json.Marshal(struct {
		Transaction string `json:"transaction"`
	}{signed})
	if err != nil {
		return err
	}
	status, resp, err := kikoDo(http.MethodPost, ready.Host, "/api/v1/driver/drive/"+url.PathEscape(ride.Offer.ID)+"/payout", body, map[string]string{
		"X-Driver-Key": ready.Key,
	})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return errors.New(ready.Host + ": " + apiError(status, resp))
	}
	dropHeld(ready.Host, ride.Offer.ID)
	return dropActiveRide(ready.Host, ride.Offer.ID)
}

func saveActiveRide(server, id string) error {
	return updateProfile(func(doc *profileFile) error {
		for _, row := range doc.Rides {
			if strings.EqualFold(row.Server, server) && row.ID == id {
				return nil
			}
		}
		doc.Rides = append(doc.Rides, savedRide{Server: server, ID: id})
		return nil
	})
}

func dropActiveRide(server, id string) error {
	return updateProfile(func(doc *profileFile) error {
		next := make([]savedRide, 0, len(doc.Rides))
		for _, row := range doc.Rides {
			if strings.EqualFold(row.Server, server) && row.ID == id {
				continue
			}
			next = append(next, row)
		}
		doc.Rides = next
		return nil
	})
}

// reloadSaved loads accepted rides recorded in the profile.
// A ride the server has already paid out is dropped. A ride that cannot
// be reached is left for the next attempt.
func reloadSaved() {
	doc, err := loadProfile()
	if err != nil {
		return
	}
	var gone []savedRide
	for _, row := range doc.Rides {
		ready, err := readyFor(row.Server)
		if err != nil {
			continue
		}
		if err := fetchHeld(ready, row.ID); errors.Is(err, errRideGone) {
			gone = append(gone, row)
		}
	}
	for _, row := range gone {
		dropHeld(row.Server, row.ID)
		_ = dropActiveRide(row.Server, row.ID)
	}
}

func fetchHeld(ready readyServer, id string) error {
	status, resp, err := kikoDo(http.MethodGet, ready.Host, "/api/v1/driver/drive/"+url.PathEscape(id), nil, map[string]string{
		"X-Driver-Key": ready.Key,
	})
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return errRideGone
	}
	if status != http.StatusOK {
		return errors.New(ready.Host + ": " + apiError(status, resp))
	}
	view, err := parseRideView(resp)
	if err != nil || view.PayTo == "" {
		return errors.New(ready.Host + ": bad ride")
	}
	switch view.Status {
	case "accepted", "paid", "paying_out":
	default:
		return errRideGone
	}
	pickup := 0.0
	if view.EtaKm != nil {
		pickup = *view.EtaKm
	}
	rememberHeld(heldFromView(ready, id, pickup, view.Status, view))
	return nil
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

// ActiveCount is the number of rides this driver has claimed.
func ActiveCount() int {
	heldMu.Lock()
	defer heldMu.Unlock()
	return len(heldOrder)
}

// ActiveRide is one claimed ride, newest first, as JSON for the home screen.
func ActiveRide(index int) (string, error) {
	heldMu.Lock()
	defer heldMu.Unlock()
	if index < 0 || index >= len(heldOrder) {
		return "", errors.New("unknown ride")
	}
	ride := held[heldOrder[len(heldOrder)-1-index]]
	now := time.Now()
	immediate := 0
	if !ride.Offer.When.After(now.Add(nowWindow)) {
		immediate = 1
	}
	paid := 0
	if ride.Paid {
		paid = 1
	}
	body, err := json.Marshal(struct {
		ID           string `json:"id"`
		Server       string `json:"server"`
		Coin         int    `json:"coin"`
		StartAddress string `json:"start_address"`
		EndAddress   string `json:"end_address"`
		PickupKm     string `json:"pickup_km"`
		DistanceKm   string `json:"distance_km"`
		Immediate    int    `json:"immediate"`
		WhenDay      string `json:"when_day"`
		WhenClock    string `json:"when_clock"`
		StartLat     string `json:"start_lat"`
		StartLon     string `json:"start_lon"`
		EndLat       string `json:"end_lat"`
		EndLon       string `json:"end_lon"`
		Fare         string `json:"fare_lovelace"`
		Base         string `json:"base_lovelace"`
		PerKm        string `json:"per_km_lovelace"`
		Paid         int    `json:"paid"`
		Status       string `json:"status"`
	}{
		ID:           ride.Offer.ID,
		Server:       ride.Offer.Server,
		Coin:         ride.Offer.Coin,
		StartAddress: ride.Offer.StartAddress,
		EndAddress:   ride.Offer.EndAddress,
		PickupKm:     formatKm(ride.Offer.PickupKm),
		DistanceKm:   formatMeters(ride.Offer.TripM),
		Immediate:    immediate,
		WhenDay:      whenDay(ride.Offer.When, now),
		WhenClock:    ride.Offer.When.Format("15:04"),
		StartLat:     strconv.FormatFloat(ride.Offer.StartLat, 'f', 6, 64),
		StartLon:     strconv.FormatFloat(ride.Offer.StartLon, 'f', 6, 64),
		EndLat:       strconv.FormatFloat(ride.Offer.EndLat, 'f', 6, 64),
		EndLon:       strconv.FormatFloat(ride.Offer.EndLon, 'f', 6, 64),
		Fare:         strconv.FormatUint(ride.Offer.Fare, 10),
		Base:         strconv.FormatUint(ride.Offer.Base, 10),
		PerKm:        strconv.FormatUint(ride.Offer.PerKm, 10),
		Paid:         paid,
		Status:       ride.Status,
	})
	if err != nil {
		return "", err
	}
	return string(body), nil
}
