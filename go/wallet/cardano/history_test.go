package cardano

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func clearHistoryCache() {
	historyMu.Lock()
	historyCache = map[string]*historyState{}
	historyMu.Unlock()
}

func TestActivityCachesKnownTransactions(t *testing.T) {
	clearHistoryCache()
	t.Cleanup(clearHistoryCache)

	var infoBodies []string
	var listBodies []string
	phase := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/address_txs", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		listBodies = append(listBodies, string(raw))
		if phase == 0 {
			_, _ = w.Write([]byte(`[{"tx_hash":"aa","block_height":10,"block_time":1700000000}]`))
			return
		}
		_, _ = w.Write([]byte(`[
			{"tx_hash":"aa","block_height":10,"block_time":1700000000},
			{"tx_hash":"bb","block_height":11,"block_time":1700001000}
		]`))
	})
	mux.HandleFunc("/tx_info", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		infoBodies = append(infoBodies, string(raw))
		if phase == 0 {
			_, _ = w.Write([]byte(`[{
				"tx_hash":"aa",
				"block_time":1700000000,
				"inputs":[{"payment_addr":{"bech32":"addr1other"},"value":"5000000"}],
				"outputs":[{"payment_addr":"addr1me","value":"2000000"}]
			}]`))
			return
		}
		_, _ = w.Write([]byte(`[{
			"tx_hash":"bb",
			"block_time":1700001000,
			"inputs":[{"payment_addr":{"bech32":"addr1me"},"value":"5000000"}],
			"outputs":[{"payment_addr":{"bech32":"addr1me"},"value":"3000000"},{"payment_addr":{"bech32":"addr1other"},"value":"1800000"}]
		}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prevList := addressTxsEndpoint
	prevInfo := txInfoEndpoint
	prevClient := balanceClient
	addressTxsEndpoint = srv.URL + "/address_txs?order=block_height.desc"
	txInfoEndpoint = srv.URL + "/tx_info"
	balanceClient = srv.Client()
	t.Cleanup(func() {
		addressTxsEndpoint = prevList
		txInfoEndpoint = prevInfo
		balanceClient = prevClient
	})

	if err := RefreshActivity("addr1me"); err != nil {
		t.Fatal(err)
	}
	if err := RefreshActivity("addr1me"); err != nil {
		t.Fatal(err)
	}
	if len(listBodies) != 1 || len(infoBodies) != 1 {
		t.Fatalf("list %d info %d", len(listBodies), len(infoBodies))
	}
	rows := Activities("addr1me")
	if len(rows) != 1 || rows[0].Direction != "Received" || rows[0].Amount != "2 ADA" {
		t.Fatalf("rows = %+v", rows)
	}
	wantWhen := time.Unix(1700000000, 0).Local().Format(activityTimeLayout)
	if rows[0].When != wantWhen {
		t.Fatalf("when = %s", rows[0].When)
	}

	historyMu.Lock()
	historyCache["addr1me"].checked = time.Now().Add(-time.Second)
	historyMu.Unlock()
	phase = 1

	if err := RefreshActivity("addr1me"); err != nil {
		t.Fatal(err)
	}
	if len(infoBodies) != 2 {
		t.Fatalf("info requests = %d", len(infoBodies))
	}
	var asked struct {
		Hashes []string `json:"_tx_hashes"`
	}
	if err := json.Unmarshal([]byte(infoBodies[1]), &asked); err != nil {
		t.Fatal(err)
	}
	if len(asked.Hashes) != 1 || asked.Hashes[0] != "bb" {
		t.Fatalf("new hashes = %v", asked.Hashes)
	}
	var listed struct {
		After int64 `json:"_after_block_height"`
	}
	if err := json.Unmarshal([]byte(listBodies[1]), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.After != 10 {
		t.Fatalf("after = %d", listed.After)
	}
	rows = Activities("addr1me")
	if len(rows) != 2 || rows[0].Direction != "Sent" || rows[0].Amount != "2 ADA" || rows[1].Direction != "Received" {
		t.Fatalf("rows = %+v", rows)
	}
}
