// Package links implements Links & Calls to Action: do a website's links
// work, and do its buttons lead anywhere? Broken links and "Get started"
// buttons pointing at "#" are typical of sites published unfinished.
package links

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/capybari-repo/capybari-core/analyzer"
	"github.com/capybari-repo/capybari-core/facts"
	"github.com/capybari-repo/capybari-core/finding"
	"github.com/capybari-repo/capybari-core/webtext"
)

//go:embed capability.yaml
var capabilityYAML []byte

var capability = analyzer.MustParseCapability(capabilityYAML)

var (
	reAnchor = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
	reHref   = regexp.MustCompile(`(?i)\bhref\s*=\s*["']([^"']*)["']`)
	reNoHref = regexp.MustCompile(`(?i)\bhref\s*=`)
	// reJSApp marks pages built by a JavaScript framework, where "#" links
	// often have click handlers the scanner cannot see.
	reJSApp = regexp.MustCompile(`(?i)id=["'](?:root|app|__next|__nuxt|svelte)["']|data-reactroot|ng-version=|data-v-[0-9a-f]{6}|__NEXT_DATA__|data-svelte-h`)
	// reCTA matches the words of a call to action: the links that matter
	// most when they go nowhere.
	reCTA = regexp.MustCompile(`(?i)^\s*(?:get started|start (?:now|free|(?:your )?(?:free )?trial)|sign ?up|register|join(?: now| free)?|buy(?: now)?|order now|purchase|subscribe|download(?: now)?|book a (?:demo|call)|request a demo|contact(?: us| sales)?|try (?:it |for )?(?:now|free)|get (?:it|access|the app)|upgrade|shop now|learn more)\b`)
)

// Analyzer implements the capability.
type Analyzer struct{}

// New returns the capability.
func New() *Analyzer { return &Analyzer{} }

// Capability implements analyzer.Analyzer.
func (*Analyzer) Capability() analyzer.Capability { return capability }

// Applies declines when no page was fetched.
func (*Analyzer) Applies(in *analyzer.Input) (bool, string) {
	var ws facts.WebSnapshot
	if ok, _ := in.Evidence.Get(facts.KeyWebSnapshot, &ws); !ok || ws.Body == "" {
		return false, "no page content was fetched"
	}
	return true, ""
}

func broken(c facts.LinkCheck) bool {
	return c.Error != "" || c.Status == 404 || c.Status == 410 || c.Status >= 500
}

type page struct {
	url, html string
	jsApp     bool
}

// Analyze implements analyzer.Analyzer.
func (*Analyzer) Analyze(_ context.Context, in *analyzer.Input) (*analyzer.Result, error) {
	var ws facts.WebSnapshot
	if _, err := in.Evidence.Get(facts.KeyWebSnapshot, &ws); err != nil {
		return nil, err
	}
	l := &facts.Links{Checked: len(ws.LinkChecks)}
	for _, c := range ws.LinkChecks {
		if broken(c) {
			l.Broken = append(l.Broken, c)
		}
	}

	ps := []page{{ws.FinalURL, ws.Body, reJSApp.MatchString(ws.Body)}}
	for _, p := range ws.Pages {
		ps = append(ps, page{p.URL, p.HTML, reJSApp.MatchString(p.HTML)})
	}
	var dead []finding.Evidence
	deadInApp := 0
	seen := map[string]bool{}
	for _, p := range ps {
		for _, m := range reAnchor.FindAllStringSubmatch(p.html, -1) {
			text := strings.Join(strings.Fields(webtext.Visible(m[2])), " ")
			if !reCTA.MatchString(text) || len(text) > 40 {
				continue
			}
			href, hasHref := "", reNoHref.MatchString(m[1])
			if h := reHref.FindStringSubmatch(m[1]); h != nil {
				href = strings.TrimSpace(h[1])
			}
			lower := strings.ToLower(href)
			if hasHref && href != "" && href != "#" && !strings.HasPrefix(lower, "javascript:") {
				continue
			}
			key := strings.ToLower(text)
			if seen[key] {
				continue
			}
			seen[key] = true
			if p.jsApp {
				deadInApp++
			}
			target := href
			if target == "" {
				target = "(no link)"
			}
			l.DeadCTAs = append(l.DeadCTAs, fmt.Sprintf("%q → %s on %s", text, target, p.url))
			dead = append(dead, finding.Evidence{Location: finding.Location{URL: p.url}, Snippet: truncate(m[0], 160), Detail: fmt.Sprintf("%q links to %s", text, target)})
		}
	}

	var fs []finding.Finding
	if len(l.Broken) > 0 {
		sev := finding.Low
		if len(l.Broken) >= 2 || l.Checked > 0 && len(l.Broken)*5 >= l.Checked {
			sev = finding.Medium
		}
		var ev []finding.Evidence
		for i, b := range l.Broken {
			if i == 10 {
				break
			}
			d := fmt.Sprintf("HTTP %d", b.Status)
			if b.Error != "" {
				d = b.Error
			}
			ev = append(ev, finding.Evidence{Location: finding.Location{URL: b.URL}, Detail: d})
		}
		fs = append(fs, finding.Finding{
			Dimension: finding.DimTrust, Category: "broken-link", Severity: sev, Confidence: finding.ConfidenceHigh,
			Title:       fmt.Sprintf("%d of %d links checked are broken", len(l.Broken), l.Checked),
			Description: "Links from the front page lead to pages that do not exist or fail. Visitors hit dead ends, and it suggests nobody has clicked through the site recently.",
			Evidence:    ev,
			Impact:      &finding.Impact{Buyer: finding.BuyerSupportCost},
			Remediation: &finding.Remediation{Summary: "Fix or remove the broken links."},
		})
	}
	if len(dead) > 0 {
		sev, conf := finding.Medium, finding.ConfidenceMedium
		if deadInApp == len(dead) {
			// JavaScript apps often use href="#" with a click handler.
			sev, conf = finding.Low, finding.ConfidenceLow
		}
		fs = append(fs, finding.Finding{
			Dimension: finding.DimTrust, Category: "dead-cta", Severity: sev, Confidence: conf,
			Title:                 fmt.Sprintf("%d call-to-action button%s lead%s nowhere", len(dead), map[bool]string{true: "", false: "s"}[len(dead) == 1], map[bool]string{true: "s", false: ""}[len(dead) == 1]),
			Description:           "Buttons such as \"Get started\" or \"Sign up\" link to \"#\" or to nothing. On a generated site this usually means the button was never wired up.",
			Evidence:              dead,
			Impact:                &finding.Impact{Buyer: finding.BuyerSupportCost},
			Remediation:           &finding.Remediation{Summary: "Point every call to action at a real page or form, or remove it."},
			FalsePositiveGuidance: "JavaScript apps sometimes open a dialog from a \"#\" link; check the button in a browser.",
		})
	}

	return &analyzer.Result{
		Findings: fs,
		Summary:  fmt.Sprintf("%d same-site link(s) checked, %d broken; %d call(s) to action leading nowhere", l.Checked, len(l.Broken), len(l.DeadCTAs)),
		Evidence: map[string]any{facts.KeyLinks: l},
	}, nil
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
