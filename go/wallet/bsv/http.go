package bsv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// WhatsOnChain uses 404 when an address has never been seen on chain.
var errNotFound = errors.New("not found")

func getJSON(op, endpoint string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("bsv: %s: %w", op, err)
	}
	req.Header.Set("Accept", "application/json")
	return doJSON(op, req)
}

func postJSON(op, endpoint string, payload []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("bsv: %s: %w", op, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return doJSON(op, req)
}

func doJSON(op string, req *http.Request) ([]byte, error) {
	resp, err := balanceClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bsv: %s: %w", op, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("bsv: %s: %w", op, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("bsv: %s: %w", op, errNotFound)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("bsv: %s: %s", op, resp.Status)
	}
	return body, nil
}

func decodeJSON(body []byte, dest any) error {
	return json.Unmarshal(body, dest)
}
