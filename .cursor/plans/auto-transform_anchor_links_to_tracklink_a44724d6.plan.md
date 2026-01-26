---
name: Auto-transform anchor links to TrackLink
overview: Add automatic transformation of plain `<a href="url">` links in templates to use `{{ TrackLink "url" . }}`, excluding template variables and links that already use TrackLink.
todos:
  - id: add-regex-pattern
    content: Add regex pattern to regTplFuncs in models/common.go to transform <a href="url"> to use TrackLink
    status: completed
  - id: test-transformations
    content: Test the transformation with various link formats (plain URLs, template variables, already-tracked links)
    status: completed
---

# Auto-transform anchor links to TrackLink

## Overview

Enhance listmonk to automatically transform plain HTML anchor links (`<a href="url">`) in templates to use the TrackLink function, making link tracking seamless for users without requiring manual TrackLink syntax.

## Current Behavior

- Users must manually add `{{ TrackLink "url" }}` or use the shorthand `url@TrackLink`
- Plain `<a href="url">` links are not automatically tracked

## Implementation

### Changes to `models/common.go`

Add a new regex pattern to the `regTplFuncs` array that:

1. Matches `<a href="url">` or `<a href='url'>` patterns (handles both single and double quotes)
2. Excludes links that already contain template syntax (`{{` or `}}`)
3. Excludes links that already use TrackLink (`@TrackLink` or `TrackLink`)
4. Transforms plain URLs to `{{ TrackLink "url" . }}`

The regex pattern should be added **before** the existing TrackLink transformations in the array, so plain links are transformed first, then existing TrackLink syntax is normalized.

### Regex Pattern Details

The pattern needs to:

- Match `<a` tag with `href` attribute containing a URL
- Handle both `href="url"` and `href='url'` formats
- Skip if href contains `{{` (template variables like `{{ UnsubscribeURL }}`)
- Skip if href contains `@TrackLink` or `TrackLink` (already tracked)
- Extract the URL and wrap it with `{{ TrackLink "url" . }}`

### Example Transformations

**Before:**

```html
<a href="https://example.com">Visit us</a>
<a href='https://example.com'>Visit us</a>
<a href="/relative/path">Internal link</a>
```

**After:**

```html
<a href="{{ TrackLink "https://example.com" . }}">Visit us</a>
<a href="{{ TrackLink "https://example.com" . }}">Visit us</a>
<a href="{{ TrackLink "/relative/path" . }}">Internal link</a>
```

**Excluded (not transformed):**

```html
<a href="{{ UnsubscribeURL }}">Unsubscribe</a>
<a href="https://example.com@TrackLink">Already tracked</a>
<a href="{{ TrackLink "https://example.com" }}">Already tracked</a>
```

### Processing Order

The transformation happens during template compilation in `models/campaigns.go`:

1. Template body is processed through `regTplFuncs` (line 161-163)
2. Campaign body is processed through `regTplFuncs` (line 182-184)
3. Alt body is processed through `regTplFuncs` (line 199-201)

The new pattern will automatically apply to all these locations.

## Files to Modify

- `models/common.go`: Add new regex pattern to `regTplFuncs` array

## Testing Considerations

After implementation, verify:

1. Plain links are transformed correctly
2. Template variables are not transformed
3. Already-tracked links are not double-transformed
4. Both single and double quotes work
5. Relative URLs are handled
6. Edge cases (empty href, malformed tags) don't break