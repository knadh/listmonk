
## OIDC Single Sign On

Listmonk supports single sign-on with OIDC (OpenID Connect). Any standards compliant OIDC provider can be configured in Settings -> Security -> OIDC

### User auto-creation
If `Settings -> Security -> OIDC -> Auto-create users` is turned on, when users login via OIDC, an account is auto-created if an existing account is not found (based on the OIDC e-mail ID).

## OIDC role mappings

When OIDC user auto-creation is enabled, listmonk can assign user and list roles based on claims present in the verified OIDC ID token.

Role mappings are evaluated only when a new OIDC user is automatically created. Existing users are not updated on subsequent logins, even if their OIDC claims change.

### Supported claim values

Role mappings support top-level OIDC claims with either of these JSON types:

- string
- array of strings

Matching is exact and case-sensitive.

Nested claims, regular expressions, glob patterns, JSONPath expressions, boolean logic, and partial matches are not supported.

For complex provider-specific claims, configure the identity provider to expose the required value as a top-level ID token claim.

### Mapping behavior

Mappings are evaluated in declaration order.

The first matching mapping wins.

A mapping can define:

- `user_role_id`
- `list_role_id`
- or both

If a matching mapping omits one of these fields, the corresponding configured default OIDC role is preserved.

If no mapping matches, the configured OIDC default user role and default list role are used.

Example:

```json
[
  {
    "claim": "groups",
    "match": "listmonk-admin",
    "user_role_id": 1
  },
  {
    "claim": "department",
    "match": "IT",
    "user_role_id": 5,
    "list_role_id": 6
  }
]
```

For an ID token such as:

```json
{
  "groups": [
    "users",
    "listmonk-admin"
  ],
  "department": "IT"
}
```

the first mapping matches and wins. The user receives `user_role_id = 1`, while the configured default list role is preserved.

If the mappings are reversed, the `department = IT` mapping wins instead.

### Managing role mappings through the API

OIDC role mappings can be managed without directly editing the database.

#### Get the current mappings

```http
GET /api/settings/oidc/role-mappings
```

The caller must have the `settings:get` permission.

Example response:

```json
{
  "data": [
    {
      "claim": "department",
      "match": "IT",
      "list_role_id": 6
    },
    {
      "claim": "groups",
      "match": "listmonk-admin",
      "user_role_id": 1
    }
  ]
}
```

#### Replace the mappings

```http
PUT /api/settings/oidc/role-mappings
Content-Type: application/json
```

The caller must have the `settings:manage` permission.

The request body is the complete ordered list of mappings:

```json
[
  {
    "claim": "department",
    "match": "IT",
    "list_role_id": 6
  },
  {
    "claim": "groups",
    "match": "listmonk-admin",
    "user_role_id": 1
  }
]
```

The API validates each mapping before saving it.

Each mapping must have:

- a non-empty `claim`
- a non-empty `match`
- at least one of `user_role_id` or `list_role_id`

Referenced role IDs must exist and must be valid for the corresponding role type.

If validation fails, the API returns an HTTP `400` response and the existing mappings are left unchanged.

After a successful update, listmonk uses its normal settings reload mechanism.

### Example with a string claim

ID token:

```json
{
  "department": "IT"
}
```

Mapping:

```json
[
  {
    "claim": "department",
    "match": "IT",
    "user_role_id": 5,
    "list_role_id": 6
  }
]
```

A newly auto-created user receives user role `5` and list role `6`.

### Example with an array claim

ID token:

```json
{
  "groups": [
    "users",
    "marketing",
    "listmonk-admin"
  ]
}
```

Mapping:

```json
[
  {
    "claim": "groups",
    "match": "listmonk-admin",
    "user_role_id": 1
  }
]
```

The mapping matches because one value in the `groups` array is exactly `listmonk-admin`.

Only the user role is overridden. The configured default list role is preserved.

### Important notes

- Role mappings apply only to newly auto-created OIDC users.
- Existing users are not resynchronized on login.
- Claim names are top-level ID token claim names.
- Matching is exact and case-sensitive.
- Array claims are supported only when all values are strings.
- The first matching mapping wins.
- If no mapping matches, the configured default OIDC roles are used.
- The identity provider must include mapped claims in the OIDC ID token.


# Tutorials

Tutorials for configuring listmonk SSO with popular OIDC providers.

## Keycloak
Keycloak configuration for listmonk SSO integration.

### 1. Create a new client in Keycloak
In the Keycloak admin, use an existing realm, or create a new realm. Create a new client in `Clients → Create`.

- **General Settings**
    - **Client type**: `OpenID Connect`
    - **Client ID**: `listmonk` (or any preferred name)
    - **Name**: Optional descriptive name (e.g., "listmonk SSO")
- **Capability Config**:
    - **Client authentication**: On
    - **Authorization**: On
    - **Authentication Flow**
        - **Standard Flow**: On
        - **Direct Access grants**: On
- **Login Settings**:
    - **Root URL**: Copy the **Redirect URL for oAuth provider** value from listmonk Admin -> Settings -> Security -> OIDC. It will look like `https://listmonk.yoursite.com/auth/oidc`
    - **Valid redirect URIs**: Same as the Root URL above
    - **Valid post logout redirect URIs**: *

After the client creation steps above, go to the client's `Credentials` tab and copy the `Client Secret`.

### 2. Configure Listmonk
2. In Listmonk Admin -> Settings -> Security -> OIDC.
    - **Enable OIDC SSO**: Turn on
    - **Provider URL**: `https://keycloak.yoursite.com/realms/{realm}` (replace `{realm}` with the chosen realm name). This URL is as of v26.3 and may differ across Keycloak versions.
    - **Provider name**: Set a name to show on the listmonk login form, eg: `Login with OrgName`
    - **Client ID**: Client ID set in Keycloak, eg: `listmonk`
    - **Client Secret**: Client Secret copied from Keycloak
    - **Auto-create users from SSO**: (Optional) Enable to automatically create users who don't exist
    - **Default user role**: (Required if auto-create enabled) Select role for new users



## Authentik  
Authentik configuration for listmonk SSO integration.

### 1. Create a new OIDC provider in Authentik
In the Authentik admin interface, create a new OIDC provider for listmonk.

- **Provider Settings**:  
    - **Name**: `listmonk` (or any preferred name)
    - **Signing Key**: `authentik Self-signed Certificate`
    - **Client Type**: `Confidential`
    - **Client ID**: `listmonk` (or any preferred name)
    - **Redirect URIs**: Copy the **Redirect URL for oAuth provider** value from listmonk Admin -> Settings -> Security -> OIDC. It will look like `https://listmonk.yoursite.com/auth/oidc`

After creating the provider, copy the **Client Secret**.

### 2. Create an application in Authentik
Create a new application and connect it to the newly created provider.

- **Application Settings**:
    - **Name**: `listmonk` (or any preferred name)
    - **Slug**: `listmonk` (or any preferred slug. Used in the redirect URL)
    - **Provider**: Select the OIDC provider created in the previous step

### 3. Configure listmonk
In listmonk Admin → Settings → Security → OIDC:

- **Enable OIDC SSO**: Turn on
- **Provider URL**: `https://authentik.yoursite.com/application/o/{slug}/` (replace `{slug}` with the application's slug)
- **Provider Name**: Set a name to show on the login form (e.g., `Login with OrgName`)
- **Client ID**: Client ID set in Authentik (e.g., `listmonk`)
- **Client Secret**: Client Secret copied from Authentik
- **Auto-create users from SSO**: (Optional) Enable to automatically create users who don't exist
- **Default user role**: (Required if auto-create enabled) Select role for new users

## Google Workspace  
Google Workspace (Google Cloud) configuration for listmonk SSO integration.

### 1. Create a new OIDC provider in Google Cloud Console / Google Workspace
In the Google Cloud Console interface, create a new Project.

- **Project Settings**:  
    - **Project name**: `Listmonk` (or any preferred name)
- **Branding Settings**:
    - **App name**: `Listmonk` (or any preferred name, this will be visible to the users.)
    - **Authorised domains**: `listmonk.example.com` (or domains that your instance is available on.)

After creating the project, goto **Clients**.

### 2. Create an client in project.
Create a new client and configure it.

- **Application Settings**:
    - **Application type**: `Web application`
    - **Name**: `listmonk` (or any preferred name)
    - **Authorised JavaScript origins**: `https://listmonk.example.com` (or domains that your instance is available on.)
    - **Authorised redirect URIs**: `https://listmonk.example.com/auth/oidc` (or domains that your instance is available on, value is also available in the Settings mentioned above. (Redirect URL for oAuth provider))

Hit save and note the Client ID and Client Secret

### 3. Configure listmonk
In listmonk Admin → Settings → Security → OIDC:

- **Enable OIDC SSO**: Turn on
- **Provider URL**: `https://accounts.google.com` (select Google to Auto-Fill)
- **Provider Name**: Set a name to show on the login form (e.g., `Login with OrgName`)
- **Client ID**: Client ID copied from Console (e.g., `XXXX.apps.googleusercontent.com`)
- **Client Secret**: Client Secret copied from Console
- **Auto-create users from SSO**: (Optional) Enable to automatically create users who don't exist
- **Default user role**: (Required if auto-create enabled) Select role for new users
