package gps

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	pureDest     = "io.github.rinigus.PureMaps"
	purePath     = "/io/github/rinigus/PureMaps"
	pureIface    = "io.github.rinigus.PureMaps"
	pureNavPath  = "/io/github/rinigus/PureMaps/navigator"
	pureNavIface = "io.github.rinigus.PureMaps.navigator"
	lipDest      = "com.jolla.lipstick"
	lipPath      = "/"
	lipIface     = "org.sailfishos.fileservice"
)

// pureBus is one session-bus connection.
type pureBus interface {
	Call(dest, path, iface, member, sig string, body []byte) error
	Close()
}

// dialPure opens the session bus. Tests replace it.
var dialPure = func() (pureBus, error) {
	return dialSession()
}

// pureWait is the pause between attempts to reach Pure Maps after opening it.
var pureWait = 500 * time.Millisecond

// Navigate shows the pickup in Pure Maps and asks it to start guidance.
// ShowPoi accepts a place. Start begins guidance only when a route is
// already loaded; otherwise the pickup stays on the map. If Pure Maps
// is not on the bus, lipstick opens the same geo: place and this waits
// until ShowPoi can be delivered.
func Navigate(lat, lon float64, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Pickup"
	}
	latArg := strconv.FormatFloat(lat, 'f', 6, 64)
	lonArg := strconv.FormatFloat(lon, 'f', 6, 64)
	bus, err := dialPure()
	if err != nil {
		return err
	}
	defer bus.Close()
	err = showPoi(bus, title, lat, lon)
	if serviceUnknown(err) {
		geo := "geo:" + latArg + "," + lonArg
		if err = bus.Call(lipDest, lipPath, lipIface, "openUrl", "s", bodyS(geo)); err != nil {
			return err
		}
		err = waitShow(bus, title, lat, lon)
	}
	if err != nil {
		return err
	}
	return bus.Call(pureDest, pureNavPath, pureNavIface, "Start", "", nil)
}

func showPoi(bus pureBus, title string, lat, lon float64) error {
	return bus.Call(pureDest, purePath, pureIface, "ShowPoi", "sdd", bodySDD(title, lat, lon))
}

func waitShow(bus pureBus, title string, lat, lon float64) error {
	var last error
	for i := 0; i < 30; i++ {
		time.Sleep(pureWait)
		last = showPoi(bus, title, lat, lon)
		if !serviceUnknown(last) {
			return last
		}
	}
	return fmt.Errorf("Pure Maps is not running")
}

func serviceUnknown(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "ServiceUnknown") || strings.Contains(msg, "NameHasNoOwner")
}

func bodyS(s string) []byte {
	return appendStr(nil, s)
}
