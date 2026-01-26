---
name: Add UTM parameters to tracked links
overview: Automatically append UTM parameters to all tracked links when redirecting, using campaign name and date for analytics tracking similar to Mailchimp.
todos:
  - id: add-utm-helper
    content: Add helper function to build UTM query string with campaign name and date
    status: pending
  - id: modify-link-redirect
    content: Modify LinkRedirect function to append UTM parameters before redirecting
    status: pending
  - id: get-campaign-name
    content: Add logic to retrieve campaign name from campaign UUID
    status: pending
---

# Add UTM parameters to tracked links

## Overview

Automatically append UTM parameters to all tracked links when users click on them. This allows destination websites to identify traffic sources from email campaigns in their analytics tools (Google Analytics, etc.), similar to how Mailchimp and other email platforms work.

## Current Behavior

- Tracked links redirect directly to the original URL without any query parameters
- Destination websites cannot identify that traffic came from email campaigns
- No UTM parameters are added for analytics tracking

## Implementation

### UTM Parameter Values

The following UTM parameters will be automatically appended to all tracked links:

- `utm_source=Tech+in+Asia+Main+List` (permanent, URL-encoded)
- `utm_medium=email` (permanent)
- `utm_campaign={CAMPAIGN_NAME}` (dynamic, from campaign name, URL-encoded)
- `utm_term={DATE}` (dynamic, current date in YYYY-MM-DD format)
- `utm_content=listmonk` (permanent)

### Changes to `cmd/public.go`

Modify the `LinkRedirect` function to:

1. Get campaign details (name) from the campaign UUID
2. Build UTM parameter string with the specified values
3. Append UTM parameters to the destination URL before redirecting
4. Handle URLs that already have query parameters (use `&` instead of `?`)

### Implementation Details

**Function: `LinkRedirect` (line 534-553)**

- After retrieving the original URL from `RegisterCampaignLinkClick`
- Fetch campaign name using `campUUID` (need to add method to get campaign by UUID)
- Build UTM query string with:
  - `utm_source=Tech+in+Asia+Main+List` (URL encoded)
  - `utm_medium=email`
  - `utm_campaign={campaign.Name}` (URL encoded)
  - `utm_term={current_date}` (YYYY-MM-DD format)
  - `utm_content=listmonk`
- Append to URL (check if URL already has query params)
- Redirect to URL with UTM parameters

### URL Encoding

Use Go's `net/url` package to properly encode:

- Campaign names with special characters
- Spaces in "Tech in Asia Main List" → "Tech+in+Asia+Main+List"
- Other special characters in campaign names

### Date Format

Use `time.Now().Format("2006-01-02")` for YTM term date in YYYY-MM-DD format.

### Edge Cases

1. **URLs with existing query parameters**: Append with `&` instead of `?`
2. **URLs with existing UTM parameters**: Override existing UTM params with new values
3. **Campaign not found**: Fallback to campaign UUID or skip UTM params
4. **URL encoding**: Properly encode all parameter values

### Example Output

**Before:**

```
https://example.com/page
```

**After:**

```
https://example.com/page?utm_source=Tech+in+Asia+Main+List&utm_medium=email&utm_campaign=Newsletter+2024&utm_term=2024-01-15&utm_content=listmonk
```

**With existing query params:**

```
https://example.com/page?id=123&utm_source=Tech+in+Asia+Main+List&utm_medium=email&utm_campaign=Newsletter+2024&utm_term=2024-01-15&utm_content=listmonk
```

## Files to Modify

- `cmd/public.go`: Modify `LinkRedirect` function to append UTM parameters
- `internal/core/campaigns.go`: Add helper method to get campaign name by UUID (if needed, or use existing GetCampaign)

## Testing Considerations

After implementation, verify:

1. UTM parameters are correctly appended to tracked links
2. Campaign names with special characters are properly URL-encoded
3. URLs with existing query parameters work correctly
4. Date format is correct (YYYY-MM-DD)
5. All permanent values are correct (source, medium, content)
6. Campaign name is correctly retrieved and encoded