
## OIDC Single Sign On

Listmonk supports single sign-on with OIDC (OpenID Connect). Any standards compliant OIDC provider can be configured in Settings -> Security -> OIDC

### User auto-creation
If `Settings -> Security -> OIDC -> Auto-create users` is turned on, when users login via OIDC, an account is auto-created if an existing account is not found (based on the OIDC e-mail ID).

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
