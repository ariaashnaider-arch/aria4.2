package main

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func Test4111StrictWebSocketHandshakeAndFailures(t *testing.T) {
	mode := "ok"
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, e := base64.StdEncoding.DecodeString(r.Header.Get("Sec-WebSocket-Key"))
		if e != nil || len(key) != 16 || r.ProtoMajor != 1 {
			w.WriteHeader(400)
			return
		}
		if mode == "forbidden" {
			w.WriteHeader(403)
			return
		}
		if mode == "missing" {
			w.WriteHeader(404)
			return
		}
		h := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		accept := base64.StdEncoding.EncodeToString(h[:])
		if mode == "wrong-accept" {
			accept = "invalid"
		}
		w.Header().Set("Upgrade", "websocket")
		w.Header().Set("Connection", "Upgrade")
		w.Header().Set("Sec-WebSocket-Accept", accept)
		w.WriteHeader(101)
	}))
	defer srv.Close()
	if e := probeWebSocket(context.Background(), srv.Client(), srv.URL, "localhost"); e != nil {
		t.Fatal(e)
	}
	for _, v := range []struct{ mode, detail string }{{"forbidden", "HTTP 403"}, {"missing", "HTTP 404"}, {"wrong-accept", "کلید تأیید"}} {
		mode = v.mode
		if e := probeWebSocket(context.Background(), srv.Client(), srv.URL, "localhost"); e == nil || !strings.Contains(e.Error(), v.detail) {
			t.Fatalf("%s: %v", mode, e)
		}
	}
}

func trialWorker4111() EdgeRoute {
	return EdgeRoute{ID: "worker-test", Name: "Worker", Provider: "cloudflare-worker", Enabled: true, Networks: []string{"ws"}, Endpoint: LinkEndpoint{Address: "trial.workers.dev", Host: "trial.workers.dev", SNI: "trial.workers.dev", Port: 443}, Probe: &EdgeProbe{At: time.Now().UnixMilli(), Failure: "websocket", Stages: []ProbeStage{{Name: "tls", OK: true}, {Name: "origin", OK: true}, {Name: "websocket", OK: false}}}}
}

func Test4111TrialDoesNotPromoteRouteOrAllowAutomaticDelivery(t *testing.T) {
	a := automaticApp(t)
	p := profileTemplates[0]
	p.Path = "/connect/fixture/"
	p.Ready = true
	a.state.Profiles = []Profile{p}
	raw := map[string]any{"id": newUUID(), "email": "test"}
	a.state.Users = []User{{Email: "test", Name: "Test", Enabled: true, Raw: raw, Links: []Connection{{Name: p.Name, Profile: p.Key, Link: shareLink(p, raw, "https://origin.example")}}}}
	a.edges.Routes = []EdgeRoute{trialWorker4111()}
	w := httptest.NewRecorder()
	a.edgeExport(w, httptest.NewRequest("GET", "/api/edges/export?email=test&route=worker-test", nil))
	if w.Code != 409 {
		t.Fatal("ordinary export accepted unverified Worker", w.Code)
	}
	w = httptest.NewRecorder()
	a.edgeExport(w, httptest.NewRequest("GET", "/api/edges/export?email=test&route=worker-test&trial=1", nil))
	var d struct {
		Trial  bool
		Links  []string
		Routes int
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &d) != nil || !d.Trial || d.Routes != 1 || len(d.Links) != 1 || !strings.Contains(d.Links[0], "trial.workers.dev") {
		t.Fatal(w.Code, w.Body.String())
	}
	if a.edges.Routes[0].Verified || len(a.sales.Notices) != 0 {
		t.Fatal("trial changed health or delivery")
	}
	a.telegram = TelegramConfig{Enabled: true, Token: "fixture-token", ChatID: 42}
	w = httptest.NewRecorder()
	a.queueEdgeDelivery(w, httptest.NewRequest("POST", "/api/edges/delivery", strings.NewReader(`{"email":"test","route":"worker-test","id":"aaaaaaaaaaaaaaaaaaaaaaaa"}`)))
	if w.Code == 200 || len(a.sales.Notices) != 0 {
		t.Fatal("unverified trial queued Telegram delivery")
	}
	w = httptest.NewRecorder()
	a.edgeExport(w, httptest.NewRequest("GET", "/api/edges/export?email=test&trial=1", nil))
	if w.Code != 400 {
		t.Fatal("trial without selected route accepted")
	}
}

func Test4111TrialRequiresFreshOriginAndTLS(t *testing.T) {
	if !edgeTrialAllowed(trialWorker4111()) {
		t.Fatal("reachable Worker denied trial")
	}
	for _, mutate := range []func(*EdgeRoute){
		func(v *EdgeRoute) { v.Enabled = false }, func(v *EdgeRoute) { v.Provider = "deno" }, func(v *EdgeRoute) { v.Probe.At = time.Now().Add(-16 * time.Minute).UnixMilli() }, func(v *EdgeRoute) { v.Probe.Stages[0].OK = false }, func(v *EdgeRoute) { v.Probe.Stages[1].OK = false }, func(v *EdgeRoute) { v.Probe.Failure = "origin" },
	} {
		v := trialWorker4111()
		mutate(&v)
		if edgeTrialAllowed(v) {
			t.Fatal("unreachable/stale route accepted")
		}
	}
}

func Test4111AllCountryAssetsAreEmbedded(t *testing.T) {
	dir, e := assets.ReadDir("web/flags")
	if e != nil || len(dir) < 250 {
		t.Fatal("incomplete country asset package", e, len(dir))
	}
	for _, v := range dir {
		b, e := assets.ReadFile("web/flags/" + v.Name())
		if e != nil || len(b) == 0 {
			t.Fatal("empty flag", v.Name(), e)
		}
	}
	a := automaticApp(t)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, httptest.NewRequest("GET", "/flags/fr.svg", nil))
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "image/svg+xml") || !strings.Contains(w.Body.String(), "flag-icons-fr") {
		t.Fatal("France image not served", w.Code)
	}
}
