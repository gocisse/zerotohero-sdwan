package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"velocloud-academy/backend/internal/content"
)

func simRouter(t *testing.T, src ContentSource) http.Handler {
	t.Helper()
	return NewRouter(Deps{Health: fakePinger{}, Content: src, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
}

func TestSimulatorEndpoints(t *testing.T) {
	course, err := content.Load(os.DirFS("../content/testdata/valid"))
	if err != nil {
		t.Fatal(err)
	}
	src := fakeContent{c: course}
	post := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		simRouter(t, src).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		return rec
	}
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		simRouter(t, src).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}
	tests := []struct {
		name         string
		rec          *httptest.ResponseRecorder
		wantCode     int
		wantContains []string
	}{
		{"list", get("/api/topologies"), 200, []string{`"id":"tiny"`, `"teaching":true`, `"edges":2`, `"warnings":[]`}},
		{"one topology", get("/api/topologies/tiny"), 200, []string{`"topology"`, `"profiles"`, `"routing.bfd"`, `"ui_bases"`, `"links_tab"`}},
		{"topology not found", get("/api/topologies/nope"), 404, []string{"not_found"}},
		{"effective config", get("/api/topologies/tiny/edges/e1/effective-config"), 200,
			[]string{`"kind":"edge-override"`, `"profile_value":true`, `"status":"verified"`, `"status":"not-modelled"`, `"evaluation_order"`}},
		{"effective config edge not found", get("/api/topologies/tiny/edges/zz/effective-config"), 404, []string{"not_found"}},
		{"effective config topology not found", get("/api/topologies/zz/edges/e1/effective-config"), 404, []string{"not_found"}},
		{"what-if override removed", post("/api/topologies/tiny/what-if", `{"type":"remove_edge_override","edge":"e1","setting":"routing.bfd"}`), 200,
			[]string{`"status":"changed"`, `"assumption 8"`, `"before_source"`}},
		{"what-if profile change", post("/api/topologies/tiny/what-if", `{"type":"set_profile_setting","profile":"p1","setting":"routing.bfd","value":false}`), 200,
			[]string{`"status":"kept-override"`, `"assumption 5"`}},
		{"what-if bad type", post("/api/topologies/tiny/what-if", `{"type":"explode"}`), 400, []string{"invalid_change"}},
		{"what-if bad value", post("/api/topologies/tiny/what-if", `{"type":"set_profile_setting","profile":"p1","setting":"routing.bfd","value":{"a":1}}`), 400, []string{"invalid_change", "single value"}},
		{"what-if bad json", post("/api/topologies/tiny/what-if", `{`), 400, []string{"bad_request"}},
		{"what-if topology not found", post("/api/topologies/zz/what-if", `{"type":"assign_profile"}`), 404, []string{"not_found"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rec.Body.String()
			if tt.rec.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d: %s", tt.rec.Code, tt.wantCode, body)
			}
			if !json.Valid([]byte(body)) {
				t.Errorf("body is not JSON: %s", body)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(body, s) {
					t.Errorf("body missing %q: %s", s, body)
				}
			}
		})
	}
}

func TestSimulatorReportsInvalidContent(t *testing.T) {
	src := fakeContent{err: content.ErrorList{{File: "topologies/x.yaml", Message: "edges[e1].profile: unknown profile \"ghost\""}}}
	for _, path := range []string{"/api/topologies", "/api/topologies/x", "/api/topologies/x/edges/e/effective-config"} {
		rec := httptest.NewRecorder()
		simRouter(t, src).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != 503 || !strings.Contains(rec.Body.String(), "content_invalid") || !strings.Contains(rec.Body.String(), "topologies/x.yaml") {
			t.Errorf("%s: code %d body %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestOverlayEndpoints(t *testing.T) {
	course, err := content.Load(os.DirFS("../content/testdata/valid"))
	if err != nil {
		t.Fatal(err)
	}
	src := fakeContent{c: course}
	post := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		simRouter(t, src).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		return rec
	}
	tests := []struct {
		name         string
		rec          *httptest.ResponseRecorder
		wantCode     int
		wantContains []string
	}{
		{"tunnels, empty body", post("/api/topologies/tiny/tunnels", ""), 200, []string{`"id":"gateway:e1:gw1"`, `"type":"dynamic"`, `"reason_code":"no_traffic_yet"`, `"status":"verified"`}},
		{"tunnels with a link down", post("/api/topologies/tiny/tunnels", `{"links_down":["l1"]}`), 200, []string{`"state":"down"`}},
		{"tunnels with dynamic traffic", post("/api/topologies/tiny/tunnels", `{"traffic":[{"a":"e1","b":"e2","first_packet_ago_s":20,"last_packet_ago_s":5}]}`), 200, []string{`"id":"dynamic:e1:e2"`, `"state":"up"`}},
		{"tunnels, silence of 180 s", post("/api/topologies/tiny/tunnels", `{"traffic":[{"a":"e1","b":"e2","first_packet_ago_s":300,"last_packet_ago_s":180}]}`), 200, []string{`"state":"torn-down"`}},
		{"tunnels, unknown link", post("/api/topologies/tiny/tunnels", `{"links_down":["nope"]}`), 400, []string{"invalid_conditions", "unknown link nope"}},
		{"tunnels, bad json", post("/api/topologies/tiny/tunnels", `{`), 400, []string{"bad_request"}},
		{"tunnels, unknown topology", post("/api/topologies/zz/tunnels", `{}`), 404, []string{"not_found"}},
		{"timeline", post("/api/topologies/tiny/timeline", `{"events":[{"at_s":5,"type":"traffic_start","a":"e1","b":"e2"},{"at_s":20,"type":"traffic_stop","a":"e1","b":"e2"}]}`), 200, []string{`"frames"`, `"first_packet"`, `"torn-down"`, `"at_s":200`}},
		{"timeline, bad event", post("/api/topologies/tiny/timeline", `{"events":[{"at_s":1,"type":"explode"}]}`), 400, []string{"invalid_events"}},
		{"timeline, unknown topology", post("/api/topologies/zz/timeline", `{}`), 404, []string{"not_found"}},
		{"paths", post("/api/topologies/tiny/paths", `{"flow":{"source":"e1","destination":{"kind":"edge","id":"e2"},"application":"voice"}}`), 200,
			[]string{`"options"`, `"chosen"`, `"reachability"`, `"assumed":true`, `"mtu"`, `"effective_min":1388`}},
		{"paths with a gateway down", post("/api/topologies/tiny/paths", `{"flow":{"source":"e1","destination":{"kind":"edge","id":"e2"}},"conditions":{"gateways_unreachable":["gw1"]}}`), 200, []string{`"usable":false`, `peer_unreachable`}},
		{"paths, bad flow", post("/api/topologies/tiny/paths", `{"flow":{"source":"nope","destination":{"kind":"edge","id":"e2"}}}`), 400, []string{"invalid_flow"}},
		{"paths, unknown topology", post("/api/topologies/zz/paths", `{}`), 404, []string{"not_found"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rec.Body.String()
			if tt.rec.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d: %s", tt.rec.Code, tt.wantCode, body)
			}
			if !json.Valid([]byte(body)) {
				t.Errorf("body is not JSON: %s", body)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(body, s) {
					t.Errorf("body missing %q: %s", s, body)
				}
			}
		})
	}
}

func TestRoutingEndpoints(t *testing.T) {
	course, err := content.Load(os.DirFS("../content/testdata/valid"))
	if err != nil {
		t.Fatal(err)
	}
	src := fakeContent{c: course}
	post := func(path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		simRouter(t, src).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)))
		return rec
	}
	tests := []struct {
		name         string
		rec          *httptest.ResponseRecorder
		wantCode     int
		wantContains []string
	}{
		{"routes, empty body uses the defaults", post("/api/topologies/tiny/routes", ""), 200,
			[]string{`"prefix":"10.9.0.0/24"`, `"source":"remote-static"`, `"source":"local-static"`, `"selected":true`, `"key":"bgp_overlay_prefix"`, `"assumption":22`, `"filter_default_action"`, `"value":"permit"`}},
		{"routes with a link down", post("/api/topologies/tiny/routes", `{"conditions":{"links_down":["l1"]}}`), 200, []string{`"reachable_code":"PR_UNREACHABLE"`}},
		{"routes with a changed assumption", post("/api/topologies/tiny/routes", `{"assumptions":{"advertise_static":false}}`), 200, []string{`"changed":true`}},
		{"routes, bad assumption", post("/api/topologies/tiny/routes", `{"assumptions":{"filter_order":"random"}}`), 400, []string{"invalid_assumptions"}},
		{"routes, bad condition", post("/api/topologies/tiny/routes", `{"conditions":{"neighbors_down":["e1/nope"]}}`), 400, []string{"invalid_conditions"}},
		{"routes, bad json", post("/api/topologies/tiny/routes", `{`), 400, []string{"bad_request"}},
		{"routes, unknown topology", post("/api/topologies/zz/routes", `{}`), 404, []string{"not_found"}},
		{"reachability", post("/api/topologies/tiny/reachability", `{"query":{"source":"e2","destination":"10.9.0.4"}}`), 200,
			[]string{`"outcome":"reachable"`, `"stages"`, `"id":"known"`, `"id":"path"`, `"lesson_ref"`, `"depends_on"`, `"assumptions"`}},
		{"reachability, unreachable names the first failing stage", post("/api/topologies/tiny/reachability", `{"query":{"source":"e2","destination":"10.9.0.4"},"assumptions":{"advertise_static":false}}`), 200,
			[]string{`"outcome":"unreachable"`, `"first_failing":"advertised"`}},
		{"reachability, unknown source", post("/api/topologies/tiny/reachability", `{"query":{"source":"zz","destination":"10.9.0.4"}}`), 400, []string{"invalid_query"}},
		{"reachability, bad destination", post("/api/topologies/tiny/reachability", `{"query":{"source":"e2","destination":"nope"}}`), 400, []string{"invalid_query"}},
		{"reachability, cross segment", post("/api/topologies/tiny/reachability", `{"query":{"source":"e2","destination":"10.9.0.4","destination_segment":"other"}}`), 200, []string{`"outcome":"not-modelled"`}},
		{"reachability, unknown topology", post("/api/topologies/zz/reachability", `{}`), 404, []string{"not_found"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rec.Body.String()
			if tt.rec.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d: %s", tt.rec.Code, tt.wantCode, body)
			}
			if !json.Valid([]byte(body)) {
				t.Errorf("body is not JSON: %s", body)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(body, s) {
					t.Errorf("body missing %q: %s", s, body)
				}
			}
		})
	}
}
