listmonk supports (>= v4.0.0) creating systems users with granular permissions
to various features, including list-specific permissions. Users can login with
a username and password, or via an OIDC (OpenID Connect) handshake if an auth
provider is connected. Users can be assigned a "user role" to grant generic
user and app permissions and a "list role" to grant per-list and per-messenger
permissions, managed per-user in the _Admin -> Users -> Users_ UI.

## User roles

A user role is a collection of generic user and app permissions, described
below. User roles can be managed in the _Admin -> Users -> User roles_ UI.

| Group       | Permission              | Description                                                                                                                                                                                                                          |
| ----------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| lists       | lists:get_all           | Get details of all lists                                                                                                                                                                                                             |
|             | lists:manage_all        | Update and delete all lists                                                                                                                                                                                                          |
|             | lists:create            | Create new lists                                                                                                                                                                                                                     |
| subscribers | subscribers:get         | Get individual subscriber details                                                                                                                                                                                                    |
|             | subscribers:get_all     | Get all subscribers and their details                                                                                                                                                                                                |
|             | subscribers:manage      | Add, update, and delete subscribers                                                                                                                                                                                                  |
|             | subscribers:import      | Import subscribers from external files                                                                                                                                                                                               |
|             | subscribers:sql_query   | Run SQL queries on subscriber data. **WARNING:** This permission will allow the querying of all lists and subscribers directly from the database with SQL expressions, superceding individual list and subscriber permissions above. |
|             | tx:send                 | Send transactional messages to subscribers                                                                                                                                                                                           |
| campaigns   | campaigns:get           | Get and view campaigns belonging to permitted lists                                                                                                                                                                                  |
|             | campaigns:get_all       | Get and view campaigns across all lists                                                                                                                                                                                              |
|             | campaigns:get_analytics | Access campaign performance metrics                                                                                                                                                                                                  |
|             | campaigns:manage        | Create, update, and delete campaigns                                                                                                                                                                                                 |
|             | messengers:get_all      | Send campaigns to any/all configured email servers and custom senders                                                                                                                                                              |
| bounces     | bounces:get             | Get email bounce records                                                                                                                                                                                                             |
|             | bounces:manage          | Process and handle bounced emails                                                                                                                                                                                                    |
|             | webhooks:post_bounce    | Receive bounce notifications via webhook                                                                                                                                                                                             |
| media       | media:get               | Get uploaded media files                                                                                                                                                                                                             |
|             | media:manage            | Upload, update, and delete media                                                                                                                                                                                                     |
| templates   | templates:get           | Get email templates                                                                                                                                                                                                                  |
|             | templates:manage        | Create, update, and delete templates                                                                                                                                                                                                 |
| users       | users:get               | Get system user accounts                                                                                                                                                                                                             |
|             | users:manage            | Create, update, and delete user accounts                                                                                                                                                                                             |
|             | roles:get               | Get user roles and permissions                                                                                                                                                                                                       |
|             | roles:manage            | Create and modify user roles                                                                                                                                                                                                         |
| settings    | settings:get            | Get system settings                                                                                                                                                                                                                  |
|             | settings:manage         | Modify system configuration                                                                                                                                                                                                          |
|             | settings:maintain       | Perform system maintenance tasks                                                                                                                                                                                                     |

## List roles

A list role is a collection of per-list and per-messenger permissions that can
be used to segment subscriber lists, messengers and campaigns to user groups.
List roles can be managed in the _Admin -> Users -> List roles_ UI.

### Subscribers

Each list can be assigned a view (read) or manage (update) permission. Users may
only access subscriber lists they have the view permission for, both within the
admin UI and via API calls. The `lists:get_all` and `lists:manage_all` user role
permissions override this behaviour, giving users access to all lists regardless.

Lists created by a user with the `lists:create` user role permission will be
added to that user's list role with both view and manage permissions granted.
If a user does not have a list role, they may be unable to view created lists.

- If you want users to be able to create lists shared with other users,
  grant them `lists:create` and put them in the same list role.
- If you want users to be able to create lists hidden from other users,
  grant them `lists:create` and put them in different list roles.
- If you do not want users to create/manage custom subscriber lists,
  do not grant them `lists:create`.

### Messengers

Each named messenger can be enabled (checked) or disabled (unchecked).
Messengers can be configured in the _Admin -> Settings -> Settings -> SMTP_
and _Admin -> Settings -> Settings -> Messengers_ UIs. Users may only send
campaigns to subscriber lists via [messengers](https://listmonk.app/docs/messengers/)
enabled by their list role. The `messengers:get_all` user role permission
overrides this behaviour, allowing a user to send from any/all messengers.

A user must be able to send from at least one messenger. If a user has not been
granted any named messengers via their list role or use role permissions, they
will default to the generic `email` messenger.

## API users

A user account can be of two types, a regular user or an API user. API users
are meant for intertacting with the listmonk APIs programmatically. Unlike
regular user accounts that have custom passwords or OIDC for authentication,
API users get an automatically generated secret token.
