package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsMiddleware(t *testing.T) {

	httpRequestsTotal.Reset()
	httpRequestDuration.Reset()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r := chi.NewRouter()
	r.Use(MetricsMiddleware())
	r.Get("/test", testHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	expectedTotal := `
		# HELP http_requests_total Total number of HTTP requests.
		# TYPE http_requests_total counter
		http_requests_total{method="GET",path="/test",status_code="OK"} 1
	`
	if err := testutil.CollectAndCompare(httpRequestsTotal, strings.NewReader(expectedTotal)); err != nil {
		t.Errorf("unexpected metrics for http_requests_total: %v", err)
	}
}
