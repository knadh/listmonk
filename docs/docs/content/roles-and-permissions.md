listmonk supports (>= v4.0.0) creating systems users with granular permissions to various features, including list-specific permissions. Users can login with a username and password, or via an OIDC (OpenID Connect) handshake if an auth provider is connected. Various permissions can be grouped into "user roles", which can be assigned to users. List-specific permissions can be grouped into "list roles".

## User roles

A user role is a collection of user related permissions. User roles are attached to user accounts. User roles can be managed in `Admin -> Users -> User roles` The permissions are described below.

| Group       | Permission              | Description                                                                                                                                                                                                                          |
| ----------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| lists       | lists:get_all           | Get details of all lists                                                                                                                                                                                                             |
|             | lists:manage_all        | Create, update, and delete all lists                                                                                                                                                                                                 |
| subscribers | subscribers:get         | Get individual subscriber details                                                                                                                                                                                                    |
|             | subscribers:get_all     | Get all subscribers and their details                                                                                                                                                                                                |
|             | subscribers:manage      | Add, update, and delete subscribers                                                                                                                                                                                                  |
|             | subscribers:import      | Import subscribers from external files                                                                                                                                                                                               |
|             | subscribers:sql_query   | Run raw SQL queries on subscriber data.<br /><span style="color: #de4a45;">**WARNING:**</span><span style="font-size: 0.875em; line-height: 1.3; color:#888;">This permission allows execution of arbitrary SQL expressions and SQL functions. While it is readonly on the table data, it allows querying of all lists and subscribers directly from the database superceding individual list and subscriber permissions. Raw SQL expressions make it possible to obtain Postgres database configuration and potentially interact with other Postgres system features. Give this permission ONLY to trusted users. [Learn more](#subscriberssql_query). |
|             | tx:send                 | Send transactional messages to subscribers                                                                                                                                                                                           |
| campaigns   | campaigns:get           | Get and view campaigns belonging to permitted lists                                                                                                                                                                                  |
|             | campaigns:get_all       | Get and view campaigns across all lists                                                                                                                                                                                              |
|             | campaigns:get_analytics | Access campaign performance metrics                                                                                                                                                                                                  |
|             | campaigns:manage        | Create, update, and delete campaigns                                                                                                                                                                                                 |
| bounces     | bounces:get             | Get email bounce records                                                                                                                                                                                                             |
|             | bounces:manage          | Process and handle bounced emails                                                                                                                                                                                                    |
|             | webhooks:post_bounce    | Receive bounce notifications via webhook                                                                                                                                                                                             |
| media       | media:get               | Get uploaded media files                                                                                                                                                                                                             |
|             | media:manage            | Upload, update, and delete media                                                                                                                                                                                                     |
| templates   | templates:get           | Get email templates                                                                                                                                                                                                                  |
|             | templates:manage        | Create, update, and delete templates                                                                                                                                                                                                 |
| users       | users:get               | Get system user accounts                                                                                                                                                                                                             |
|             | users:manage            | Create, update, and delete user accounts <span style="color: #de4a45;">**WARNING:**</span><span style="font-size: 0.875em; line-height: 1.3; color:#888;">This permission allows creation of users with any role, including Super Admin. This permission should only be given to Super Admin level accounts</span>                              |
|             | roles:get               | Get user roles and permissions                                                                                                                                                                                                       |
|             | roles:manage            | Create and modify user roles                                                                                                                                                                                                         |
| settings    | settings:get            | Get system settings                                                                                                                                                                                                                  |
|             | settings:manage         | Modify system configuration                                                                                                                                                                                                          |
|             | settings:maintain       | Perform system maintenance tasks                                                                                                                                                                                                     |

## List roles

A list role is a collection of permissions assigned per list. Each list can be assigned a view (read) or manage (update) permission. List roles are attached to user accounts. Only the lists defined in a list role is accessible by the user, be it on the admin UI or via API calls. Do note that the `lists:get_all` and `lists:manage_all` permissions in user roles override all per-list permissions.

## API users

A user account can be of two types, a regular user or an API user. API users are meant for intertacting with the listmonk APIs programmatically. Unlike regular user accounts that have custom passwords or OIDC for authentication, API users get an automatically generated secret token.

## `subscribers:sql_query`

This permission allowers users to write and execute arbitrary SQL queries on the database. Although it is executed as a read-only transaction disallowing changing of data in the database tables, it allows querying of all lists, subscribers and other data directly from the database superceding individual list and subscriber permissions.

Raw SQL expressions also make it possible to obtain Postgres database configuration and potentially interact with other Postgres system features. Give this permission ONLY to trusted users.

If this permission is being assigned to many users, it is highly recommended that you create a custom Postgres role disallowing any privileged operations. For example:

```sql
CREATE ROLE listmonk_app WITH
    LOGIN
    PASSWORD '...'
    NOSUPERUSER
    NOCREATEDB
    NOCREATEROLE
    NOREPLICATION;
```
