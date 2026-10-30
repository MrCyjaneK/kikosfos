package wallet

import (
	"errors"
	"fmt"
	"sync"

	"kiko-sfos/gps"
)

var (
	gpsMu   sync.Mutex
	gpsSrc  gps.Source
	gpsKind string
)

// GPSDriver is the selected backend. An empty profile value is GeoClue.
func GPSDriver() (string, error) {
	doc, err := loadProfile()
	if err != nil {
		return "", err
	}
	return gps.Normalize(doc.GPS), nil
}

// SetGPSDriver stores the backend. Unknown names become GeoClue.
// The live source is closed when the choice changes and opened on the next read.
func SetGPSDriver(kind string) error {
	kind = gps.Normalize(kind)
	if err := updateProfile(func(doc *profileFile) error {
		doc.GPS = kind
		return nil
	}); err != nil {
		return err
	}
	closeLater(takeGPS(kind))
	return nil
}

// GPSPosition is "lat\nlon\naccuracy" in degrees and meters.
func GPSPosition() (string, error) {
	kind, err := GPSDriver()
	if err != nil {
		return "", err
	}
	fix := useGPS(kind).Snapshot()
	if !fix.HasPos {
		if fix.Err != "" {
			return "", errors.New(fix.Err)
		}
		return "", errors.New(gps.NoFix)
	}
	return fmt.Sprintf("%.6f\n%.6f\n%.0f", fix.Lat, fix.Lon, fix.AccM), nil
}

func useGPS(kind string) gps.Source {
	kind = gps.Normalize(kind)
	gpsMu.Lock()
	if gpsSrc != nil && gpsKind == kind {
		src := gpsSrc
		gpsMu.Unlock()
		return src
	}
	old := gpsSrc
	gpsSrc = gps.Open(kind)
	gpsKind = kind
	src := gpsSrc
	gpsMu.Unlock()
	closeLater(old)
	return src
}

// ReleaseGPS drops the live fix. The saved driver is left as it is.
func ReleaseGPS() {
	closeLater(takeGPS(""))
}

// takeGPS detaches the live source when keep is empty or names a different driver.
func takeGPS(keep string) gps.Source {
	gpsMu.Lock()
	defer gpsMu.Unlock()
	if gpsSrc == nil || (keep != "" && gpsKind == keep) {
		return nil
	}
	src := gpsSrc
	gpsSrc = nil
	gpsKind = ""
	return src
}

func closeLater(src gps.Source) {
	if src != nil {
		go src.Close()
	}
}

func resetGPS() {
	gpsMu.Lock()
	defer gpsMu.Unlock()
	if gpsSrc != nil {
		gpsSrc.Close()
		gpsSrc = nil
	}
	gpsKind = ""
}
