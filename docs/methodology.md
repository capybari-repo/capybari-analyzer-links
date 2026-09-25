# Methodology: Links & Calls to Action (experimental)

## Inputs

- `link_checks` from the Website Snapshot: the status of each same-site link followed from the front page (the pages read with GET, then up to 15 more with HEAD, falling back to GET without reading the body when HEAD is not supported). Robots.txt and the crawler's exclusions (login, logout, admin, cart, checkout, register, feeds, files) apply.
- The HTML of the pages read.

## Rules

| Rule | Trigger | Severity | Confidence |
|---|---|---|---|
| `broken-link` | status 404, 410, ≥ 500, or a network error | low; medium if ≥ 2 or ≥ 20% of those checked | high |
| `dead-cta` | an anchor whose text starts with a call to action (get started, start free trial, sign up, register, join, buy, order now, purchase, subscribe, download, book/request a demo, contact, try now/free, get it/access/the app, upgrade, shop now, learn more; at most 40 characters) and whose `href` is missing, empty, `#` or `javascript:` | medium on static pages; low when every one is on a JavaScript-framework page (React/Vue/Next/Nuxt/Svelte/Angular markers) | medium / low |

Each distinct call-to-action text is reported once. Anchors like `#faq` (in-page navigation) are not dead.

## Limitations

External links are not checked. Buttons that are `<button>` elements, not links, are not judged, because their behaviour depends on scripts.
