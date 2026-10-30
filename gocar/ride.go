package main

import (
	"strings"
)

// ride is one offer or one claimed trip. Active rides arrive as the home
// screen's JSON. Offers are filled from the KIKO_drive_* getters.
type ride struct {
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
	FareLovelace string `json:"fare_lovelace"`
	Paid         int    `json:"paid"`
	Status       string `json:"status"`
	Claimed      bool   `json:"-"`
	FareAmount   string `json:"-"`
	FareEuro     string `json:"-"`
}

func (r *ride) normalize() {
	if r.Status == "" && r.Paid == 1 {
		r.Status = "paid"
	} else if r.Status == "" && r.Claimed {
		r.Status = "accepted"
	}
}

func (r ride) whenText() string {
	if r.Immediate == 1 {
		return "Now"
	}
	s := strings.TrimSpace(r.WhenDay + " " + r.WhenClock)
	if s == "" {
		return "Scheduled"
	}
	return s
}

func (r ride) fareText() string {
	if r.FareEuro != "" {
		return r.FareEuro
	}
	if r.FareAmount != "" {
		return r.FareAmount
	}
	return r.FareLovelace
}

func (r ride) statusText() string {
	switch r.Status {
	case "paid_out":
		return "Paid out"
	case "paid":
		return "Paid"
	case "accepted":
		return "Accepted"
	default:
		if r.Claimed {
			return "Accepted"
		}
		return ""
	}
}

// primary is the one high-importance action for this state.
func (r ride) primary() (string, bool) {
	if !r.Claimed {
		return "Accept", true
	}
	if r.Paid == 0 {
		return "Check payment", true
	}
	if r.Status != "paid_out" {
		return "Cash out", true
	}
	return "", false
}

// quoteParts reads KIKO_wallet_quote: amount, lovelace, euro, optional error.
func quoteParts(coin int, lovelace, raw string) (amount, euro string) {
	if lovelace == "" {
		return "", ""
	}
	if coin < 0 {
		return lovelace, ""
	}
	parts := strings.Split(raw, "\n")
	if len(parts) > 3 && parts[3] != "" {
		return lovelace, ""
	}
	if len(parts) > 0 {
		amount = parts[0]
	}
	if len(parts) > 2 {
		euro = parts[2]
	}
	if amount == "" {
		amount = lovelace
	}
	return amount, euro
}

func wrapLines(s string, width int) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) == 0 || width < 1 {
		return ""
	}
	var lines []string
	cur := ""
	for _, w := range fields {
		if cur == "" {
			cur = w
			continue
		}
		if len([]rune(cur))+1+len([]rune(w)) > width {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur += " " + w
	}
	lines = append(lines, cur)
	return strings.Join(lines, "\n")
}

func ridesKey(active, offers []ride) string {
	var b strings.Builder
	for _, r := range active {
		b.WriteString(r.ID)
		b.WriteByte(':')
		b.WriteString(r.Status)
		b.WriteByte(':')
		b.WriteString(r.fareText())
		b.WriteByte('|')
	}
	b.WriteByte('#')
	for _, r := range offers {
		b.WriteString(r.ID)
		b.WriteByte(':')
		b.WriteString(r.fareText())
		b.WriteByte('|')
	}
	return b.String()
}
