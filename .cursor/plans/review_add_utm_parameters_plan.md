# PR-style review: Add UTM parameters to tracked links

**Reviewer stance:** Last maintainer, merge only if plan is correct, maintainable, and safe for upstream.

---

## 1. What the plan gets right

- **Place of change:** `LinkRedirect` in [cmd/public.go](cmd/public.go) (lines 534–552) is the right hook. Redirect happens after `RegisterCampaignLinkClick` returns the original URL; that’s where UTM must be applied.
- **Campaign by UUID:** `a.core.GetCampaign(0, campUUID, "")` is already used in the same file ([cmd/public.go](cmd/public.go) line 151). No new core method needed; the plan’s “if needed, or use existing GetCampaign” is correct—use existing.
- **Existing query params:** Plan correctly calls out using `&` when the URL already has `?`.
- **URL encoding:** Using `net/url` for encoding is correct.
- **Date:** `time.Now().Format("2006-01-02")` is correct for YYYY-MM-DD.
- **Campaign not found:** Handling missing campaign (fallback/skip UTM) is the right behavior so the redirect still works.

---

## 2. Blockers for upstream (must fix)

### 2.1 Hardcoded tenant-specific values

Plan specifies:

- `utm_source=Tech+in+Asia+Main+List`
- `utm_content=listmonk`

Listmonk is multi-tenant/open-source. Hardcoding a specific brand and product name is not acceptable in core code.

**Required:** Make these configurable. Options:

- Add an optional config block, e.g. `[app.utm]` or `[tracking.utm]` in config.toml:
  - `utm_source` (optional; if empty, skip adding UTM or use a neutral default)
  - `utm_medium` (default `email`)
  - `utm_content` (optional)
  - `utm_enabled` (default `false` so existing deployments are unchanged)
- When `utm_enabled` is false or source is not set, do not append UTM (preserve current behavior).
- Document that these are for analytics and often brand-specific.

Without this, the feature cannot be merged as-is for upstream.

### 2.2 Typo

- Line 75: “YTM term” → should be **“UTM term”**.

---

## 3. Edge cases the plan should specify

### 3.1 URL with fragment (`#`)

Example: `https://example.com/page#section`. Appending `?utm_...` must not put parameters inside the fragment. Correct behavior:

- Parse URL, append query to the path+query part (before `#`), then re-append fragment.
- Use `net/url` (e.g. `url.Parse` → set `RawQuery` or `Query()` then `String()`), not naive string concat with `?`/`&`.

### 3.2 “Override existing UTM parameters”

Plan says: “Override existing UTM params with new values.” That implies:

- Parse the destination URL’s query string.
- Set or replace the UTM keys (`utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`).
- Rebuild the URL.

So implementation must be “parse → set query params → rebuild URL”, not “append `&utm_...`” (which would duplicate existing UTM keys). The plan should state this explicitly to avoid a buggy “append-only” implementation.

### 3.3 Campaign name safe for query string

Campaign names are arbitrary (admin-set). They must be:

- Treated as opaque string and **query-escaped** (e.g. `url.QueryEscape` or building `url.Values` and `Encode()`).
- Plan already says “URL encode”; worth stressing that this applies to campaign name and any other dynamic value.

### 3.4 Empty or invalid URL from `RegisterCampaignLinkClick`

Today we do `c.Redirect(http.StatusTemporaryRedirect, url)`. If we later build `urlWithUTM`:

- If the original `url` is empty or invalid, redirecting to a UTM-modified URL must not make things worse (e.g. parse and only append UTM when the result is still a valid absolute URL). Document or implement: on parse failure, redirect to original `url` unchanged.

---

## 4. Performance and layering

- **Extra DB call:** For every link click we would add `GetCampaign(0, campUUID, "")` to get the campaign name. That’s one more query per click.
- **Alternatives:**  
  - Extend the `register-link-click` query to return both `url` and campaign name (e.g. join `campaigns` and return `campaigns.name`) so we keep a single DB round-trip. Plan says “Modify LinkRedirect” and “get campaign details (name)”; it doesn’t mandate where the name comes from. Returning name from the same query is preferable for a hot path.  
- If we keep a separate GetCampaign call, that’s acceptable but should be an explicit trade-off in the plan (one extra query per redirect).

---

## 5. Testing and behavior

Plan’s testing list is good. Add:

- URL with fragment: UTM params appear before `#`, fragment preserved.
- URL that already has UTM: our values override; no duplicate keys.
- Campaign not found: redirect still happens, either without UTM or with fallback (camp UUID), and no 5xx.
- UTM disabled or not configured: behavior identical to current (no UTM, no extra logic on redirect path if possible).

---

## 6. Summary table

| Item | Verdict |
|------|--------|
| Hook (LinkRedirect) | Correct |
| Use existing GetCampaign(0, campUUID, "") | Correct; no new core method required |
| Hardcoded utm_source / utm_content | **Blocker** – must be configurable and off by default |
| YTM typo | Fix to UTM |
| Fragment (#) handling | Must specify: parse URL, add params before fragment |
| Override existing UTM | Must specify: parse query, set params, rebuild (no append-only) |
| Campaign name encoding | Reiterate: always query-encode |
| Extra DB round-trip | Prefer returning campaign name from register-link-click; else document trade-off |
| Core vs cmd | No change to internal/core required if we only use existing GetCampaign |

---

## 7. Recommended implementation outline

1. **Config:** Add optional `[app.utm]` (or similar): `enabled`, `source`, `medium`, `content`. Default `enabled=false`. Only append UTM when enabled and source (and any required fields) are set.
2. **LinkRedirect (cmd/public.go):**  
   - Call `RegisterCampaignLinkClick` as today.  
   - If UTM enabled: get campaign name (either from existing `GetCampaign(0, campUUID, "")` or from extended register-link-click query). On error (e.g. campaign not found), use fallback (e.g. camp UUID or skip UTM).  
   - Build UTM query (using `url.Values` and `Encode()`; date and campaign name from config/campaign).  
   - Parse destination URL with `url.Parse`; set `RawQuery` (or merge and set Query) so that our UTM params override existing UTM; ensure fragment stays in `Fragment` and is re-applied.  
   - Redirect to the resulting URL. If parsing fails, redirect to original `url`.
3. **Optional optimization:** Extend [queries/links.sql](queries/links.sql) `register-link-click` to return campaign name in the same statement and use that in Core/App to avoid a second DB call.
4. **Docs:** In plan or PR description, state: UTM is optional, config-driven, and safe for fragment/query/encoding; when disabled, behavior is unchanged.

With the blocker (configurable, off-by-default UTM) and the edge-case clarifications above, the plan is in good shape for implementation and merge.
