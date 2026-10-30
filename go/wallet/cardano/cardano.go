// Package cardano is wallet index 0: one Shelley account on mainnet.
package cardano

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"sync"

	"github.com/blinklabs-io/bursa"
)

const (
	Name = "Cardano"
	ID   = "cardano"
)

// Account is a mainnet CIP-1852 wallet at the default indexes.
// Address receives ADA. PrivateKey is the payment extended signing key (addr_xsk).
type Account struct {
	Mnemonic     string
	Address      string
	StakeAddress string
	PrivateKey   string
}

// Create generates a 24-word mainnet account.
func Create() (Account, error) {
	mnemonic, err := bursa.GenerateMnemonic()
	if err != nil {
		return Account{}, fmt.Errorf("cardano: generate mnemonic: %w", err)
	}
	return Restore(mnemonic)
}

// Restore rebuilds the mainnet account from a BIP39 recovery phrase.
// The BIP39 passphrase is empty.
func Restore(mnemonic string) (Account, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	created, err := bursa.NewWallet(mnemonic, bursa.WithNetwork("mainnet"))
	if err != nil {
		return Account{}, fmt.Errorf("cardano: %w", err)
	}
	return Account{
		Mnemonic:     mnemonic,
		Address:      created.PaymentAddress,
		StakeAddress: created.StakeAddress,
		PrivateKey:   created.PaymentExtendedSKey.String(),
	}, nil
}

// Send is wired for the generic wallet API and does not build or broadcast.
func Send(string, string) error {
	return errors.New("sending is not available")
}

// IconPNG is a small mark the wallet list can draw.
func IconPNG() []byte {
	iconOnce.Do(func() {
		iconPNG = drawIcon()
	})
	return iconPNG
}

var (
	iconOnce sync.Once
	iconPNG  []byte
)

func drawIcon() []byte {
	const n = 96
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	blue := color.RGBA{R: 0x00, G: 0x33, B: 0xAD, A: 0xFF}
	white := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			img.SetRGBA(x, y, blue)
		}
	}
	cx, cy, r := n/2, n/2, n/2-14
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.SetRGBA(x, y, white)
			}
		}
	}
	r2 := r / 2
	for y := cy - r2; y <= cy+r2; y++ {
		for x := cx - r2; x <= cx+r2; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2*r2 {
				img.SetRGBA(x, y, blue)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
