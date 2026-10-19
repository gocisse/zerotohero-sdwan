package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"velocloud-academy/backend/internal/content"
)

type fakeContent struct {
	c   *content.Course
	err error
}

func (f fakeContent) Current() (*content.Course, error) { return f.c, f.err }

func TestContentEndpoints(t *testing.T) {
	course, err := content.Load(os.DirFS("../content/testdata/valid"))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name         string
		src          fakeContent
		path         string
		wantCode     int
		wantContains []string
		wantAbsent   []string
	}{
		{"course", fakeContent{c: course}, "/api/course", 200, []string{`"title":"Test Course"`, `"id":"first"`, `"question_count":2`}, nil},
		{"lesson", fakeContent{c: course}, "/api/lessons/second", 200, []string{`"section_id":"intro"`, "horizontal rule above", `"prerequisites":["first"]`}, nil},
		{"diagram", fakeContent{c: course}, "/api/diagrams/net", 200, []string{`"id":"net"`, `"route":{"nodes":["h1","sw","h2"],"links":["l1","l2"]}`, `"highlight":[]`}, nil},
		{"diagram not found", fakeContent{c: course}, "/api/diagrams/nope", 404, []string{"not_found"}, nil},
		{"glossary", fakeContent{c: course}, "/api/glossary", 200, []string{`"term":"Widget"`, `"aliases":["gizmo"]`, `"related":["doohickey"]`, `"lesson_ref":"first"`}, nil},
		{"lesson not found", fakeContent{c: course}, "/api/lessons/nope", 404, []string{"not_found"}, nil},
		{"quiz hides answers", fakeContent{c: course}, "/api/sections/intro/quiz", 200,
			[]string{`"id":"q-single"`, `"text":"A"`, `"id":"true"`},
			[]string{"correct", "explanation", "lesson_ref", "Because.", "B is wrong"}},
		{"quiz section not found", fakeContent{c: course}, "/api/sections/nope/quiz", 404, []string{"not_found"}, nil},
		{"invalid content lists every error", fakeContent{err: content.ErrorList{
			{File: "a.yaml", Message: "first problem"}, {File: "b.mdx", Message: "second problem"},
		}}, "/api/course", 503, []string{"content_invalid", "a.yaml", "first problem", "b.mdx", "second problem"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewRouter(Deps{Health: fakePinger{}, Content: tt.src, Log: log}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			body := rec.Body.String()
			if rec.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d; body %s", rec.Code, tt.wantCode, body)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(body, s) {
					t.Errorf("body missing %q: %s", s, body)
				}
			}
			for _, s := range tt.wantAbsent {
				if strings.Contains(body, s) {
					t.Errorf("body must not contain %q: %s", s, body)
				}
			}
		})
	}
}
