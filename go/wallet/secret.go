package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	seedMagic = "KSED"
	pinMagic  = "KPIN"
	canary    = "kiko-pin-ok"
)

func seal(key, plaintext []byte) (nonce, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, gcm.Seal(nil, nonce, plaintext, nil), nil
}

func open(key, nonce, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func encodeSeed(key []byte, mnemonic string) ([]byte, error) {
	nonce, ciphertext, err := seal(key, []byte(mnemonic))
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, 5+len(nonce)+len(ciphertext))
	out = append(out, seedMagic...)
	out = append(out, 1)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func decodeSeed(key, blob []byte) (string, error) {
	if len(blob) < 5+12+16 || string(blob[:4]) != seedMagic || blob[4] != 1 {
		return "", errors.New("seed file is corrupt")
	}
	nonce := blob[5:17]
	plain, err := open(key, nonce, blob[17:])
	if err != nil {
		return "", errors.New("could not decrypt seed")
	}
	return string(plain), nil
}

func encodePIN(p KDFParams, salt, key []byte) ([]byte, error) {
	if len(salt) != 16 {
		return nil, errors.New("pin salt must be 16 bytes")
	}
	nonce, ciphertext, err := seal(key, []byte(canary))
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, 5+12+16+len(nonce)+len(ciphertext))
	out = append(out, pinMagic...)
	out = append(out, 1)
	var nums [12]byte
	binary.LittleEndian.PutUint32(nums[0:4], p.Time)
	binary.LittleEndian.PutUint32(nums[4:8], p.Memory)
	binary.LittleEndian.PutUint32(nums[8:12], uint32(p.Threads))
	out = append(out, nums[:]...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func decodePIN(blob []byte) (p KDFParams, salt, nonce, ciphertext []byte, err error) {
	const header = 5 + 12 + 16 + 12
	if len(blob) < header+16 || string(blob[:4]) != pinMagic || blob[4] != 1 {
		return KDFParams{}, nil, nil, nil, errors.New("pin file is corrupt")
	}
	p.Time = binary.LittleEndian.Uint32(blob[5:9])
	p.Memory = binary.LittleEndian.Uint32(blob[9:13])
	threads := binary.LittleEndian.Uint32(blob[13:17])
	if threads == 0 || threads > 255 {
		return KDFParams{}, nil, nil, nil, errors.New("pin file is corrupt")
	}
	p.Threads = uint8(threads)
	salt = append([]byte(nil), blob[17:33]...)
	nonce = append([]byte(nil), blob[33:45]...)
	ciphertext = append([]byte(nil), blob[45:]...)
	return p, salt, nonce, ciphertext, nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("random: %w", err)
	}
	return b, nil
}
