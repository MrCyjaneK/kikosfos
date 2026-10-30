package bsv

import (
	"net/http/httptest"
	"testing"
)

func useChain(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prevBase := apiBase
	prevClient := balanceClient
	prevRate := eurRateEndpoint
	apiBase = srv.URL
	balanceClient = srv.Client()
	t.Cleanup(func() {
		apiBase = prevBase
		balanceClient = prevClient
		eurRateEndpoint = prevRate
	})
}
