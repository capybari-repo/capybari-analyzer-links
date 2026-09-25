# capybari-analyzer-links (experimental)

**Capybari Source Intelligence: Links & Calls to Action: do the links and buttons go anywhere?**

- **Broken links:** the Website Snapshot records the status of every same-site link it follows from the front page. That covers the up to 5 pages it reads plus a HEAD request (headers only) for up to 15 more, following the same rules: robots.txt is honoured, and login, logout, admin, cart and file links are never requested. Links that return 404/410, a server error, or fail are reported.
- **Dead calls to action:** "Get started", "Sign up", "Buy now", "Book a demo", "Contact" and similar links that point to `#`, `javascript:` or nowhere. They are medium severity in static HTML, and low in JavaScript-framework pages, where `#` links often have click handlers.

| Finding | For buyers |
|---|---|
| `broken-link` (low; medium if 2+ or 20%+ of the links checked) | support cost |
| `dead-cta` | support cost |

It makes no requests of its own and feeds the **Finish** question of the verdict.

| | |
|---|---|
| Requires | `web-snapshot` |
| Provides | `links` evidence |
| Network | none |

Methodology: [docs/methodology.md](docs/methodology.md). License: Apache-2.0.
