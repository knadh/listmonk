# Plan validation: Auto-transform anchor links to TrackLink

## 1. Is there existing "auto transform anchor to TrackLink" logic? (No reinvention)

**Finding: There is no existing logic that transforms plain `<a href="url">` into `{{ TrackLink "url" . }}`.**

What exists today in [models/common.go](models/common.go) (`regTplFuncs`):

1. **Normalize existing TrackLink** – `{{ TrackLink "url" }}` → `{{ TrackLink "url" . }}` (add dot context).
2. **Normalize shorthand** – `https://example.com@TrackLink` → `{{ TrackLink "url" . }}`.
3. **Normalize other tags** – e.g. `{{ TrackView }}` → `{{ TrackView . }}`.

None of these touch plain `<a href="...">` links. The plan’s new behavior (detect plain anchors and wrap the URL in TrackLink) is new and does not duplicate anything.

---

## 2. Is the implementation plan correct?

**Overall: Yes.** File, hook points, and ordering are correct.

### File and hook

- **File to modify:** `models/common.go` – correct. Only `regTplFuncs` needs to change.
- **Where it applies:** The plan says template body, campaign body, and alt body are processed through `regTplFuncs`. In [models/campaigns.go](models/campaigns.go):
  - **Subject:** lines 141–144  
  - **Template body:** lines 161–163 (`body := c.TemplateBody` then `regTplFuncs`)  
  - **Campaign body:** lines 182–184 (`body = c.Body` then `regTplFuncs`)  
  - **Alt body:** lines 199–201  
  - **ConvertContent:** lines 216–218  

So all the right strings already go through `regTplFuncs`; no changes needed in `campaigns.go`. Adding a new entry to `regTplFuncs` is enough.

### Order in `regTplFuncs`

The plan says the new pattern must be added **before** the existing TrackLink-related entries. That is correct:

1. **First:** New rule: `<a href="url">` → `<a href="{{ TrackLink "url" . }}">` (plain anchors).
2. **Then:** Existing rules that normalize `{{ TrackLink "url" }}` and `url@TrackLink` to `{{ TrackLink "url" . }}`.

So the new entry should be inserted at the **beginning** of the `regTplFuncs` slice (index 0).

### Exclusions (template vars and already-tracked links)

The plan correctly requires:

- **Do not** transform when `href` contains template syntax (`{{` / `}}`), e.g. `{{ UnsubscribeURL }}`.
- **Do not** transform when the link is already tracked (`TrackLink`, `@TrackLink`).

Because `ReplaceAllString` does one fixed replacement per match, exclusion has to be done in the **regex**: the pattern should only match when the `href` value does **not** contain `{{`, `}}`, `TrackLink`, or `@TrackLink`. That implies using a restricted match (e.g. a capture group that only matches “safe” href values). A typical approach is negative lookaheads so the matched URL part never starts those substrings (e.g. `(?:.(?!{{)(?!}})(?!TrackLink)(?!@TrackLink))*`-style logic—exact pattern to be chosen in implementation). The plan does not give the exact regex; implementation will need to encode these exclusions in the pattern.

### Quotes and relative URLs

- Supporting both `href="url"` and `href='url'` needs either two patterns or one pattern that handles both quote characters; the plan is right to call this out.
- Relative URLs (e.g. `href="/path"`) are fine as long as the regex captures any non-quote (or allowed) characters; no special case needed beyond “capture the attribute value”.

---

## 3. Edge cases and testing

The plan’s testing list is good:

- Plain links transformed; template variables and already-tracked links not transformed; no double transform; single and double quotes; relative URLs.
- **Empty href:** The regex should avoid matching `href=""` or `href=''` if you don’t want to output `TrackLink ""`. E.g. require at least one character in the captured URL.
- **Malformed tags:** A regex that expects a clear `href="..."` or `href='...'` will not match malformed markup, so it won’t change it and won’t “break” anything.

---

## 4. Plan todos

The plan marks both todos as **completed**. In the current codebase, **no** new regex for `<a href="...">` exists in `regTplFuncs`. So either:

- The implementation was done in another branch/repo and the plan was copied here, or  
- The todos were marked complete by mistake.

If the feature is not yet in this repo, implement it by adding one (or two) new `regTplFunc` entries at the **start** of `regTplFuncs` in [models/common.go](models/common.go), with the exclusion and quoting behavior above.

---

## 5. Summary

| Question | Answer |
|----------|--------|
| Existing auto anchor→TrackLink? | No. No duplication. |
| Correct file? | Yes. `models/common.go` only. |
| Correct application points? | Yes. All relevant bodies and subject already use `regTplFuncs`. |
| Correct order? | Yes. New pattern must be first in `regTplFuncs`. |
| Exclusions feasible? | Yes. Must be done in the regex (e.g. negative lookaheads / restricted capture). |
| Plan implementation-ready? | Yes, once the exact regex(es) for anchor(s) and exclusions are defined. |

The plan is **valid and implementation-ready**. No changes to `models/campaigns.go` are required; only `regTplFuncs` in `models/common.go` needs a new pattern (and possibly a second one for single-quoted `href` if not combined into one).
