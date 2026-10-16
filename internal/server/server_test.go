package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuard(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := guard([]string{"127.0.0.1:5000"}, ok)
	cases := []struct {
		method, host, ctype string
		want                int
	}{
		{"GET", "127.0.0.1:5000", "", http.StatusOK},
		{"GET", "evil.example:5000", "", http.StatusForbidden},
		{"POST", "127.0.0.1:5000", "application/json", http.StatusOK},
		{"POST", "127.0.0.1:5000", "text/plain", http.StatusForbidden},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, "/api/window", strings.NewReader("{}"))
		r.Host = c.host
		r.Header.Set("Content-Type", c.ctype)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s %s %q = %d, want %d", c.method, c.host, c.ctype, w.Code, c.want)
		}
	}
}
