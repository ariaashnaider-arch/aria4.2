package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
)

// Optional browser integration fixture uses only generated test credentials and
// a controlled engine. It never connects to a real hosting account or Telegram.
func Test4111BrowserFlow(t *testing.T) {
	if os.Getenv("ARIA_UI_QA") != "1" {
		t.Skip("optional browser verification")
	}
	a := automaticApp(t)
	install := func() {
		p := profileTemplates[0]
		p.Path = "/connect/fixture/"
		p.Ready = true
		raw := map[string]any{"id": newUUID(), "email": "qa-user"}
		a.mu.Lock()
		a.state.Ready = true
		a.state.Profiles = []Profile{p}
		a.state.Users = []User{{Email: "qa-user", Name: "Test User", Enabled: true, Status: "active", Quota: 1000000000, Raw: raw, Links: []Connection{{Name: p.Name, Profile: p.Key, Link: shareLink(p, raw, "https://origin.example")}}}}
		a.mu.Unlock()
		a.geoMu.Lock()
		a.geo.Exit = GeoInfo{Code: "FR", Country: "France", Flag: "🇫🇷", IP: "8.8.8.8"}
		a.geoMu.Unlock()
		a.edgesMu.Lock()
		a.edges.Routes = []EdgeRoute{trialWorker4111()}
		a.edgesMu.Unlock()
	}
	install()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.ServeHTTP(w, r)
		if r.URL.Path == "/api/login" && r.Method == "POST" {
			install()
		}
	}))
	defer srv.Close()
	cmd := exec.Command("node", "scripts/qa4111.cjs", srv.URL)
	cmd.Env = os.Environ()
	b, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("browser: %v\n%s", e, b)
	}
	t.Log(string(b))
}
