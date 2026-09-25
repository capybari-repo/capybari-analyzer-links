package links_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	links "github.com/capybari-repo/capybari-analyzer-links"
	"github.com/capybari-repo/capybari-core/analyzer"
	"github.com/capybari-repo/capybari-core/analyzertest"
	"github.com/capybari-repo/capybari-core/facts"
	"github.com/capybari-repo/capybari-core/finding"
	"github.com/capybari-repo/capybari-schemas"
	"gopkg.in/yaml.v3"
)

func TestCapabilityMetadata(t *testing.T) {
	b, err := os.ReadFile("capability.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := analyzer.ParseCapability(b); err != nil {
		t.Fatal(err)
	}
	var doc any
	yaml.Unmarshal(b, &doc)
	if err := schemas.ValidateValue("capability.schema.json", doc); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, pages map[string]string) (facts.Links, map[string]finding.Finding, []string) {
	t.Helper()
	var mu = make(chan struct{}, 1)
	var hits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu <- struct{}{}
		hits = append(hits, r.Method+" "+r.URL.Path)
		<-mu
		body, ok := pages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	r := analyzertest.Run(t, links.New(), analyzertest.Website(srv.URL), analyzertest.Options{Online: true})
	l := analyzertest.Fact[facts.Links](t, r, facts.KeyLinks)
	got := map[string]finding.Finding{}
	for _, f := range r.Findings {
		got[f.Category] = f
	}
	return l, got, hits
}

const page = `<html><body><p>Content page with enough words to be a page.</p></body></html>`

func TestBrokenLinksAndDeadButtons(t *testing.T) {
	front := `<html><body><nav>`
	for _, p := range []string{"/a", "/b", "/c", "/d", "/e", "/f", "/missing", "/g", "/logout"} {
		front += `<a href="` + p + `">` + strings.TrimPrefix(p, "/") + `</a> `
	}
	front += `</nav><a href="#" class="btn">Get started</a> <a class="btn">Book a demo</a> <a href="/a">Learn more</a> <a href="#faq">FAQ</a></body></html>`
	l, got, hits := run(t, map[string]string{"/": front, "/a": page, "/b": page, "/c": page, "/d": page, "/e": page, "/f": page, "/g": page, "/logout": page})
	if l.Checked != 8 || len(l.Broken) != 1 || !strings.HasSuffix(l.Broken[0].URL, "/missing") || l.Broken[0].Status != 404 {
		t.Fatalf("link checks: %+v", l)
	}
	for _, h := range hits {
		if strings.Contains(h, "/logout") {
			t.Fatal("logout must never be requested")
		}
	}
	var headed bool
	for _, h := range hits {
		headed = headed || strings.HasPrefix(h, "HEAD ")
	}
	if !headed {
		t.Fatalf("links beyond the pages read are checked with HEAD: %v", hits)
	}
	if f := got["broken-link"]; f.Title != "1 of 8 links checked are broken" {
		t.Fatalf("broken-link: %+v", got)
	}
	d := got["dead-cta"]
	if d.Severity != finding.Medium || len(d.Evidence) != 2 || !strings.Contains(d.Evidence[0].Detail, `"Get started" links to #`) {
		t.Fatalf("dead CTAs (static HTML is medium; working Learn more and #faq anchors are fine): %+v", d)
	}
}

func TestDeadButtonInJSAppIsLowConfidence(t *testing.T) {
	_, got, _ := run(t, map[string]string{"/": `<html><body><div id="root"><a href="#">Sign up</a></div></body></html>`})
	if d := got["dead-cta"]; d.Severity != finding.Low || d.Confidence != finding.ConfidenceLow {
		t.Fatalf("framework apps use # links with handlers: %+v", got)
	}
}

func TestHealthySite(t *testing.T) {
	l, got, _ := run(t, map[string]string{"/": `<html><body><a href="/a">About</a> <a href="/signup">Sign up</a></body></html>`, "/a": page, "/signup": page})
	if len(got) != 0 || l.Checked != 2 {
		t.Fatalf("healthy site: %+v %+v", got, l)
	}
}
