package wallet

import "golang.org/x/crypto/argon2"

// KDFParams are the Argon2id settings stored with a PIN verifier.
type KDFParams struct {
	Time    uint32
	Memory  uint32
	Threads uint8
}

var ProductionKDF = KDFParams{
	Time:    10,
	Memory:  128 * 1024,
	Threads: 4,
}

var activeKDF = ProductionKDF

// SetKDFParams selects the parameters used the next time a PIN is created.
// Unlock uses the parameters stored with that PIN.
func SetKDFParams(p KDFParams) {
	activeKDF = p
}

func deriveKey(pin string, salt []byte, p KDFParams) []byte {
	return argon2.IDKey([]byte(pin), salt, p.Time, p.Memory, p.Threads, 32)
}
