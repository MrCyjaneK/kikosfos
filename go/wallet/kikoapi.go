package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// kikoClient talks to a Kiko ride server (kiko-ride).
var kikoClient = &http.Client{Timeout: 20 * time.Second}

// apiBase is the origin for a stored host. Tests point every host at an
// httptest server. Loopback with a port is http so a local kiko-ride on
// :8080 works; every other host is https. The driver key is a bearer
// credential, so it is not sent in the clear except to loopback.
var apiBase = defaultAPIBase

func defaultAPIBase(host string) string {
	if name, _, err := net.SplitHostPort(host); err == nil && (name == "localhost" || name == "127.0.0.1") {
		return "http://" + host
	}
	return "https://" + host
}

func kikoDo(method, host, path string, payload []byte, header map[string]string) (int, []byte, error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, apiBase(host)+path, reader)
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range header {
		req.Header.Set(key, value)
	}
	resp, err := kikoClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", host, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", host, err)
	}
	return resp.StatusCode, body, nil
}

// driverMessage is what a driver signs: a fixed prefix, the timestamp, the
// method, the request target, and the raw body, separated by newlines.
func driverMessage(timestamp int64, method, target string, body []byte) []byte {
	head := "kiko-driver-request\n" + strconv.FormatInt(timestamp, 10) + "\n" + method + "\n" + target + "\n"
	return append([]byte(head), body...)
}

// driverDo sends a driver request signed with the account key of coin.
// sendKey adds X-Driver-Key. Registration omits it and signs the key in the body.
func driverDo(method, host, path string, payload []byte, coin int, sendKey bool) (int, []byte, error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, apiBase(host)+path, reader)
	if err != nil {
		return 0, nil, err
	}
	target := req.URL.RequestURI()
	ts := time.Now().Unix()
	sig, err := signDriver(coin, driverMessage(ts, method, target, payload))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if sendKey {
		key, err := WalletMultisigMaterial(coin)
		if err != nil {
			return 0, nil, err
		}
		req.Header.Set("X-Driver-Key", key)
	}
	req.Header.Set("X-Driver-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("X-Driver-Signature", sig)
	resp, err := kikoClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", host, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("%s: %w", host, err)
	}
	return resp.StatusCode, body, nil
}

func signDriver(coin int, msg []byte) (string, error) {
	c, err := coinAt(coin)
	if err != nil {
		return "", err
	}
	mnemonic, err := WalletMnemonic(coin)
	if err != nil {
		return "", err
	}
	return c.SignMessage(mnemonic, msg)
}

// apiError is the message from a Kiko error object, or the status code.
func apiError(status int, body []byte) string {
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil && payload.Error != "" {
		return payload.Error
	}
	return "HTTP " + strconv.Itoa(status)
}

// errorObject is a JSON error body for registrationFailure.
// A body that already carries "error" is kept.
func errorObject(status int, body []byte) []byte {
	msg := apiError(status, body)
	if msg != "HTTP "+strconv.Itoa(status) {
		return body
	}
	return []byte(fmt.Sprintf(`{"error":%q}`, msg))
}
