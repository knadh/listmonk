
## OIDC Single Sign On

Listmonk supports single sign-on with OIDC (OpenID Connect). Any standards compliant OIDC provider can be configured in Settings -> Security -> OIDC

### User auto-creation
If `Settings -> Security -> OIDC -> Auto-create users` is turned on, when users login via OIDC, an account is auto-created if an existing account is not found (based on the OIDC e-mail ID).

### OIDC role mapping

Auto-created users can receive user and list roles based on top-level claims in the
verified OIDC ID token. Roles are assigned only when the user is created; subsequent
logins do not change an existing user's roles. Claims from the UserInfo endpoint
are not used for role mapping.

Configure mappings under `Settings -> Security -> OIDC -> Role mapping`, or set
the `roles` array in `security.oidc` using the settings API. For example:

```json
"roles": [
    {"claim": "groups", "match": "editors", "user_role_id": 2, "list_role_id": 3},
    {"claim": "department", "match": "marketing", "user_role_id": 4}
]
```

Replace the example role IDs with existing user and list role IDs. Each mapping
requires a non-empty `claim` and `match`; both role IDs are optional. A claim must be
a string equal to `match`, or an array containing only strings with at least one
equal to `match`. Matching is exact and case-sensitive; nested claim paths are not
supported.

Mappings are evaluated in order and the first match wins. Omitted role IDs retain
their configured defaults. If no mapping matches, both defaults apply. A default
user role is still required when auto-creation is enabled.

`PUT /api/settings` preserves mappings when `roles` is omitted or `null`; send
`"roles": []` to clear them. `PUT /api/settings/security.oidc` replaces the entire
OIDC configuration, so include all its settings, including the client secret and
any mappings to retain. Both endpoints validate mapping role IDs.

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
