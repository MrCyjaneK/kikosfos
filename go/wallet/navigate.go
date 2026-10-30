package wallet

import (
	"strconv"
	"strings"

	"kiko-sfos/gps"
)

// StartNavigation shows the pickup in Pure Maps and asks it to start guidance.
func StartNavigation(latText, lonText, title string) error {
	lat, err := strconv.ParseFloat(strings.TrimSpace(latText), 64)
	if err != nil {
		return err
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(lonText), 64)
	if err != nil {
		return err
	}
	return gps.Navigate(lat, lon, title)
}
