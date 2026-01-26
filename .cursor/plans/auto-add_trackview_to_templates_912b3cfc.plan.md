---
name: Auto-add TrackView to templates
overview: Automatically insert `{{ TrackView }}` into campaign templates when they are created or updated, placing it before the `</body>` tag if it doesn't already exist.
todos:
  - id: add-ensure-trackview
    content: Add ensureTrackView helper function to cmd/templates.go
    status: pending
  - id: integrate-create
    content: Integrate ensureTrackView into CreateTemplate function
    status: pending
  - id: integrate-update
    content: Integrate ensureTrackView into UpdateTemplate function
    status: pending
---

# Auto-add TrackView to templates

## Overview

Automatically insert `{{ TrackView }}` into campaign templates (both `campaign` and `campaign_visual` types) when they are created or updated. The tracking pixel should be placed before the `</body>` tag, and only added if it doesn't already exist in the template.

## Current Behavior

- Users must manually add `{{ TrackView }}` to templates
- It's recommended to place it in the template footer, ideally before `</body>`
- Missing `{{ TrackView }}` means email opens won't be tracked

## Implementation

### Changes to `cmd/templates.go`

Add a helper function `ensureTrackView` that:

1. Checks if the template type is `campaign` or `campaign_visual`
2. Checks if `{{ TrackView }}` already exists in the template body
3. If not present, inserts it before the `</body>` tag
4. If no `</body>` tag exists, appends it at the end of the body

The function should be called in both `CreateTemplate` and `UpdateTemplate` functions, right after validation and before compilation.

### Function Logic

```go
func ensureTrackView(tpl *models.Template) {
    // Only process campaign templates
    if tpl.Type != models.TemplateTypeCampaign && tpl.Type != models.TemplateTypeCampaignVisual {
        return
    }
    
    // Skip if TrackView already exists
    if strings.Contains(tpl.Body, "{{ TrackView }}") || 
       strings.Contains(tpl.Body, "{{TrackView}}") ||
       strings.Contains(tpl.Body, "TrackView") {
        return
    }
    
    // Insert before </body> tag if it exists
    if strings.Contains(tpl.Body, "</body>") {
        tpl.Body = strings.Replace(tpl.Body, "</body>", "{{ TrackView }}\n</body>", 1)
    } else {
        // Append at the end if no </body> tag
        tpl.Body = tpl.Body + "\n{{ TrackView }}"
    }
}
```

### Integration Points

1. **CreateTemplate function** (line 108-145):

   - Call `ensureTrackView(&o)` after validation (line 115) and before compilation (line 128)

2. **UpdateTemplate function** (line 148-186):

   - Call `ensureTrackView(&o)` after validation (line 154) and before compilation (line 168)

### Edge Cases

- Templates without `</body>` tag: Append `{{ TrackView }}` at the end
- Templates with multiple `</body>` tags: Replace only the first occurrence
- Templates with whitespace variations: Check for both `{{ TrackView }}` and `{{TrackView}}`
- Visual templates: The body is HTML, so the same logic applies

## Files to Modify

- `cmd/templates.go`: Add `ensureTrackView` helper function and integrate it into `CreateTemplate` and `UpdateTemplate`

## Testing Considerations

After implementation, verify:

1. New campaign templates automatically get `{{ TrackView }}` before `</body>`
2. New campaign_visual templates automatically get `{{ TrackView }}`
3. Templates that already have `{{ TrackView }}` are not modified
4. Templates without `</body>` tag get `{{ TrackView }}` appended
5. Transactional templates (tx type) are not modified
6. Updated templates also get `{{ TrackView }}` if missing