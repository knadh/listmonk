
## OIDC Single Sign On

Listmonk supports single sign-on with OIDC (OpenID Connect). Any standards compliant OIDC provider can be configured in Settings -> Security -> OIDC

### Redirect URL
**Redirect URL for OAuth provider** is the callback URL that has to be registered with the provider as an allowed redirect URI. It defaults to the Root URL (Settings → General) followed by `/auth/oidc`, eg: `https://listmonk.yoursite.com/auth/oidc`.

The redirect URL has to be on the same hostname that the admin login page is opened on, as the login can only be completed with the cookie set by the login page. If the admin is accessed on a different hostname than the Root URL, for instance, when the Root URL is a public hostname used in links sent to subscribers and the admin is only reachable on a private hostname, set the redirect URL to the admin hostname (eg: `https://listmonk-admin.yoursite.com/auth/oidc`) and register that with the provider.

### E-mail verification
Users logging in via OIDC are matched to listmonk accounts by the e-mail address in the OIDC claims. When **Verify e-mail** is turned on, a login is only allowed if the provider asserts that the e-mail address is verified with the standard `email_verified` claim set to `true`. If the claim is not in the ID token, listmonk requests it from the provider's UserInfo endpoint. If the claim is missing there too, or is `false`, the login is rejected with the error `OIDC email is not verified`.

This option is turned on by default on new installations. On installations upgraded from v6 or earlier, which did not check this claim, it is turned off.

Some providers, for example, Microsoft Entra ID, do not issue the `email_verified` claim at all. To use such a provider, turn off **Verify e-mail**. Only do this if the provider is trusted to control the e-mail addresses of its users, as anyone who can sign in to the provider with an e-mail address can log in to the listmonk account that has that e-mail address.

### Logout
Enable **OIDC RP-initiated logout** in Settings → Security to redirect to the provider's logout endpoint when logging out of an OIDC session, if the provider supports OIDC logout. This option is disabled by default. Register `https://listmonk.yoursite.com/admin/login` (including any configured root URL path) as an allowed post-logout redirect URI with the provider.

### User auto-creation
If `Settings -> Security -> OIDC -> Auto-create users` is turned on, when users login via OIDC, an account is auto-created if an existing account is not found (based on the OIDC e-mail ID). New users are assigned the **Default user role** and the optional **Default list role**, unless a [claim-based role mapping](#claim-based-role-mapping) matches. **Default user role** is required when either auto-creation or [role sync](#role-sync) is turned on.

### Claim-based role mapping
User roles and list roles can be assigned based on the claims in the ID token issued by the provider, for instance, group memberships in a `groups` claim. Each mapping in `Settings -> Security -> OIDC -> Claim-based role mapping` has a claim name, a value to match, and a user role and/or a list role to assign.

- A mapping matches if the value of the claim is exactly equal to the configured value (case-sensitive). If the claim is an array of strings, for instance, a list of groups, the mapping matches if any one of the items is equal to the value. Claims of any other type, or arrays that contain non-string items, never match.
- The claim name has to be a top-level claim in the ID token. Nested claims (eg: `realm_access.roles`) cannot be referenced.
- Only the claims in the ID token are evaluated. Claims that are only returned by the provider's UserInfo endpoint are ignored, so the provider has to be configured to include the claim in the ID token.
- Mappings are evaluated in the order they are listed, and only the first matching mapping is applied. Mappings to more privileged roles should be listed first.
- The roles in the matching mapping are assigned instead of the default roles. If the mapping only has a user role or only a list role, the default is assigned for the other. If no mapping matches, the default roles are assigned.

For example, with the following mappings, a user whose ID token has the claim `"groups": ["staff", "newsletter-editors"]` is assigned the `Editor` user role and the `Newsletters` list role. A user who is in neither group is assigned the default roles.

| Claim    | Value                | User role | List role     |
| -------- | -------------------- | --------- | ------------- |
| `groups` | `listmonk-admins`    | `Admin`   |               |
| `groups` | `newsletter-editors` | `Editor`  | `Newsletters` |

Role mappings are applied when a user is auto-created, and on every OIDC login if [role sync](#role-sync) is turned on.

### Role sync
By default, roles are only assigned when a user is auto-created. Later changes to the user's claims on the provider, for instance, being added to or removed from a group, are not reflected in listmonk. Turn on **Auto-sync roles on every login** to re-evaluate the [role mappings](#claim-based-role-mapping) on every OIDC login and update the user's roles. This applies to every user who logs in via OIDC, including existing accounts that were not auto-created.

With role sync turned on, the provider becomes the source of truth for the roles of users who log in via OIDC:

- On every OIDC login, the user's user role and list role are replaced with the roles from the matching mapping, or the default roles if no mapping matches. Any changes made to the user's roles in listmonk are overwritten on the next login.
- If neither the matching mapping nor the defaults have a list role, the user's list role is removed.
- Roles are only synced on login. Changes on the provider take effect on the user's next OIDC login.
- A user who is removed from all mapped groups on the provider is assigned the default roles. It does not disable the user. To revoke access, disable the user in listmonk or on the provider.
- There has to be at least one enabled Super Admin user at all times. If role sync assigns a different role to the only enabled Super Admin user, that user's login fails with an error. Ensure that such a user is mapped to the Super Admin role, or that there is another enabled Super Admin user.

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
