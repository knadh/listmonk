---
name: Visual query builder for campaign filters
overview: Create a visual query builder UI that allows non-technical users to build subscriber filters using forms and dropdowns, which generates SQL queries behind the scenes. This makes segmentation accessible without requiring SQL knowledge.
todos:
  - id: filter-builder-ui
    content: Create FilterBuilder.vue component with visual condition builder
    status: pending
  - id: condition-row
    content: Create ConditionRow.vue for individual filter conditions
    status: pending
  - id: sql-generator
    content: Implement SQL generation from visual filters in internal/core/filters.go
    status: pending
  - id: attributes-api
    content: Add API endpoint to discover available subscriber attributes
    status: pending
  - id: filter-validation
    content: Add filter validation and sanitization in cmd/campaigns.go
    status: pending
  - id: integrate-ui
    content: Integrate filter builder into campaign form UI
    status: pending
---

# Visual query builder for campaign filters

## Overview

Build a visual query builder interface that allows non-technical users to create subscriber filters using intuitive forms, dropdowns, and condition builders. The visual builder generates SQL queries behind the scenes, making segmentation accessible to all users while maintaining the power of SQL-based filtering.

## Current Challenge

- SQL queries are powerful but intimidating for non-engineers
- Need to balance ease-of-use with flexibility
- Should support both visual and advanced (SQL) modes

## Implementation Strategy

### Phase 1: Visual Query Builder UI

#### Component Structure

**1. Filter Builder Component (`frontend/src/components/FilterBuilder.vue`)**

A visual interface with:

- **Condition Groups**: AND/OR logic between groups
- **Individual Conditions**: Each condition has:
                                - **Field selector**: Dropdown of available fields (email, name, status, attributes)
                                - **Operator**: Dropdown (equals, contains, greater than, less than, etc.)
                                - **Value input**: Text, number, date picker, or dropdown based on field type
                                - **Remove button**: Delete condition

**2. Field Types and Operators**

**Standard Fields:**

- Email: `equals`, `contains`, `starts with`, `ends with`, `is not`
- Name: `equals`, `contains`, `starts with`, `is not`
- Status: `equals` (dropdown: enabled, disabled, blocklisted)
- Created Date: `before`, `after`, `between`, `equals`
- Updated Date: `before`, `after`, `between`, `equals`

**Attribute Fields (Dynamic):**

- Text attributes: `equals`, `contains`, `is not`, `is empty`, `is not empty`
- Number attributes: `equals`, `greater than`, `less than`, `between`
- Boolean attributes: `equals` (true/false)
- Array attributes: `contains`, `does not contain`

**3. UI Layout**

```
┌─────────────────────────────────────────┐
│ Filter Subscribers                      │
├─────────────────────────────────────────┤
│ [Match ALL conditions] [Match ANY]      │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ Field: [Email ▼]                   │ │
│ │ Operator: [Contains ▼]              │ │
│ │ Value: [____________] [×]           │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ [+ Add Condition]                       │
│                                         │
│ [Advanced: Show SQL]                    │
│ ┌─────────────────────────────────────┐ │
│ │ subscribers.email LIKE '%@gmail.com'│ │
│ └─────────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

### Phase 2: SQL Generation Logic

#### Backend Helper (`internal/core/filters.go`)

Create a function to convert visual filter JSON to SQL:

```go
type FilterCondition struct {
    Field    string      `json:"field"`
    Operator string      `json:"operator"`
    Value    interface{} `json:"value"`
}

type FilterGroup struct {
    Logic      string            `json:"logic"` // "AND" or "OR"
    Conditions []FilterCondition `json:"conditions"`
}

func BuildSQLFromFilters(groups []FilterGroup) (string, error) {
    // Convert visual filters to SQL WHERE clause
}
```

#### SQL Generation Rules

**Field Mapping:**

- `email` → `subscribers.email`
- `name` → `subscribers.name`
- `status` → `subscribers.status`
- `created_at` → `subscribers.created_at`
- `updated_at` → `subscribers.updated_at`
- `attribs.city` → `subscribers.attribs->>'city'`
- `attribs.age` → `(subscribers.attribs->>'age')::INT`

**Operator Mapping:**

- `equals` → `=`
- `contains` → `LIKE '%value%'`
- `starts_with` → `LIKE 'value%'`
- `ends_with` → `LIKE '%value'`
- `is_not` → `!=`
- `greater_than` → `>`
- `less_than` → `<`
- `between` → `BETWEEN value1 AND value2`
- `is_empty` → `IS NULL OR = ''`
- `is_not_empty` → `IS NOT NULL AND != ''`

### Phase 3: Attribute Discovery

#### Auto-detect Available Attributes

**Backend API (`cmd/subscribers.go`)**

- New endpoint: `GET /api/subscribers/attributes`
- Returns list of all unique attribute keys from subscriber database
- Includes type detection (string, number, boolean, array)

**Frontend Integration:**

- Fetch available attributes on component mount
- Populate field dropdown with discovered attributes
- Show appropriate input type based on attribute type

### Phase 4: Pre-built Filter Templates

#### Common Filter Presets

Create reusable filter templates:

- **Active Subscribers**: `status = 'enabled'`
- **New Subscribers (Last 30 days)**: `created_at > NOW() - INTERVAL '30 days'`
- **Subscribers with Email Domain**: `email LIKE '%@domain.com'`
- **Subscribers by City**: `attribs->>'city' = 'Singapore'`
- **Premium Subscribers**: `attribs->>'tier' = 'premium'`

Users can:

- Select a template
- Customize it
- Save as new template

### Phase 5: Hybrid Mode

#### Toggle Between Visual and SQL

- **Visual Mode** (default): Form-based builder
- **Advanced Mode**: Direct SQL editor
- **Sync**: Changes in visual mode update SQL, SQL changes can be parsed back to visual (if valid)

### Implementation Details

#### Frontend Components

**1. FilterBuilder.vue**

- Main component with condition groups
- Handles adding/removing conditions
- Validates inputs
- Generates filter JSON

**2. ConditionRow.vue**

- Individual condition row
- Field, operator, value inputs
- Type-aware value inputs

**3. FilterPreview.vue**

- Shows estimated subscriber count
- Preview of SQL query (collapsible)
- Validation errors

#### Backend Changes

**1. Filter Validation (`cmd/campaigns.go`)**

- Validate filter JSON structure
- Sanitize SQL to prevent injection
- Test query execution (dry run)

**2. Filter Execution (`internal/core/campaigns.go`)**

- Apply filters when fetching campaign subscribers
- Combine with list-based filtering
- Optimize queries for performance

### User Experience Flow

```
1. User opens campaign form
2. Clicks "Add Filters" or "Filter Subscribers"
3. Sees visual filter builder
4. Adds conditions:
   - Selects "Email" field
   - Chooses "Contains" operator
   - Enters "@gmail.com"
5. Adds another condition:
   - Selects "City" attribute
   - Chooses "Equals" operator
   - Enters "Singapore"
6. Chooses "Match ALL" (AND logic)
7. Sees preview: "~1,234 subscribers match"
8. Sees generated SQL (collapsible)
9. Saves campaign with filters
```

### Security Considerations

- **SQL Injection Prevention**: 
                                - Never allow direct SQL input in visual mode
                                - Sanitize all user inputs
                                - Use parameterized queries
                                - Whitelist allowed fields and operators

- **Query Validation**:
                                - Validate field names against schema
                                - Limit query complexity (max conditions)
                                - Test queries before execution
                                - Set query timeout limits

## Files to Create/Modify

### Backend

- `internal/core/filters.go`: Filter to SQL conversion logic
- `cmd/subscribers.go`: Add attributes discovery endpoint
- `cmd/campaigns.go`: Add filter validation and processing
- `internal/core/campaigns.go`: Apply filters in subscriber queries

### Frontend

- `frontend/src/components/FilterBuilder.vue`: Main visual builder
- `frontend/src/components/ConditionRow.vue`: Individual condition component
- `frontend/src/components/FilterPreview.vue`: Preview and SQL display
- `frontend/src/views/Campaign.vue`: Integrate filter builder
- `frontend/src/api/index.js`: Add filter-related API calls

## Migration Path

1. **Phase 1**: Start with SQL-only (current plan)
2. **Phase 2**: Add visual builder alongside SQL
3. **Phase 3**: Make visual builder default, SQL as advanced option
4. **Phase 4**: Add templates and presets

## Testing Considerations

- Visual builder generates correct SQL
- SQL queries execute safely
- Attribute discovery works correctly
- Filter combinations (AND/OR) work properly
- Performance with complex filters
- Edge cases (empty values, special characters)
- SQL injection attempts are blocked