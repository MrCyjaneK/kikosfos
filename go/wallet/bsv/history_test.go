package bsv

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/p2pkh"
)

func clearHistoryCache() {
	historyMu.Lock()
	historyCache = map[string]*historyState{}
	historyMu.Unlock()
}

func TestActivityUnknownAddressIsEmpty(t *testing.T) {
	clearHistoryCache()
	t.Cleanup(clearHistoryCache)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.NotFound(w, r)
	}))
	defer srv.Close()
	useChain(t, srv)

	if err := RefreshActivity(abandonAddress); err != nil {
		t.Fatal(err)
	}
	if rows := Activities(abandonAddress); len(rows) != 0 {
		t.Fatalf("rows = %+v", rows)
	}
	if err := RefreshActivity(abandonAddress); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("requests = %d", hits)
	}
}

func TestActivityCachesKnownTransactions(t *testing.T) {
	clearHistoryCache()
	t.Cleanup(clearHistoryCache)

	dest, _, _ := derivePayment(t, abandonMnemonic, 1)
	fundID, fundHex := payTx(t, abandonAddress, 50_000_000)
	spendID, spendHex := spendTx(t, fundID, abandonAddress, dest, 50_000_000, 40_000_000)
	hexes := map[string]string{fundID: fundHex, spendID: spendHex}
	heights := map[string]int64{fundID: 10, spendID: 11}
	times := map[string]int64{fundID: 1700000000, spendID: 1700001000}

	var listHits int
	var txPosts [][]string
	phase := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/history"):
			listHits++
			rows := []map[string]any{
				{"tx_hash": strings.Repeat("ab", 32), "height": 0},
			}
			if phase == 0 {
				rows = append(rows, map[string]any{"tx_hash": fundID, "height": heights[fundID]})
			} else {
				rows = append(rows,
					map[string]any{"tx_hash": fundID, "height": heights[fundID]},
					map[string]any{"tx_hash": spendID, "height": heights[spendID]},
				)
			}
			_ = json.NewEncoder(w).Encode(rows)
		case r.URL.Path == "/txs":
			raw, _ := io.ReadAll(r.Body)
			var asked struct {
				TxIDs []string `json:"txids"`
			}
			if err := json.Unmarshal(raw, &asked); err != nil {
				t.Errorf("txs body: %v", err)
			}
			txPosts = append(txPosts, asked.TxIDs)
			rows := make([]map[string]any, 0, len(asked.TxIDs))
			for _, id := range asked.TxIDs {
				rows = append(rows, map[string]any{
					"txid":        id,
					"hex":         hexes[id],
					"blockheight": heights[id],
					"blocktime":   times[id],
				})
			}
			_ = json.NewEncoder(w).Encode(rows)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	useChain(t, srv)

	if err := RefreshActivity(abandonAddress); err != nil {
		t.Fatal(err)
	}
	if err := RefreshActivity(abandonAddress); err != nil {
		t.Fatal(err)
	}
	if listHits != 1 || len(txPosts) != 1 {
		t.Fatalf("list %d txs %d", listHits, len(txPosts))
	}
	rows := Activities(abandonAddress)
	if len(rows) != 1 || rows[0].Direction != "Received" || rows[0].Amount != "0.5 BSV" {
		t.Fatalf("rows = %+v", rows)
	}
	wantWhen := time.Unix(1700000000, 0).Local().Format(activityTimeLayout)
	if rows[0].When != wantWhen {
		t.Fatalf("when = %s", rows[0].When)
	}

	historyMu.Lock()
	historyCache[abandonAddress].checked = time.Now().Add(-time.Second)
	historyMu.Unlock()
	phase = 1

	if err := RefreshActivity(abandonAddress); err != nil {
		t.Fatal(err)
	}
	if len(txPosts) < 2 {
		t.Fatalf("tx posts = %d", len(txPosts))
	}
	if len(txPosts[1]) != 1 || txPosts[1][0] != spendID {
		t.Fatalf("new hashes = %v", txPosts[1])
	}
	rows = Activities(abandonAddress)
	if len(rows) != 2 || rows[0].Direction != "Sent" || rows[0].Amount != "0.5 BSV" || rows[1].Direction != "Received" {
		t.Fatalf("rows = %+v", rows)
	}
}

func payTx(t *testing.T, to string, satoshis uint64) (string, string) {
	t.Helper()
	tx := transaction.NewTransaction()
	if err := tx.AddInputFrom(strings.Repeat("00", 32), 0, mustLock(t, to), satoshis, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.PayToAddress(to, satoshis); err != nil {
		t.Fatal(err)
	}
	return tx.TxID().String(), tx.Hex()
}

func spendTx(t *testing.T, prev, from, to string, inValue, outValue uint64) (string, string) {
	t.Helper()
	tx := transaction.NewTransaction()
	if err := tx.AddInputFrom(prev, 0, mustLock(t, from), inValue, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.PayToAddress(to, outValue); err != nil {
		t.Fatal(err)
	}
	return tx.TxID().String(), tx.Hex()
}

func mustLock(t *testing.T, address string) string {
	t.Helper()
	addr, err := script.NewAddressFromString(address)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := p2pkh.Lock(addr)
	if err != nil {
		t.Fatal(err)
	}
	return lock.String()
}
