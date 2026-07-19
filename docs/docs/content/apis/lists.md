# API / Lists

## Overview

The Lists API allows you to manage your mailing lists. Each list can have optional per-list settings for customization.

### Confirmation Email Configuration

For lists with double opt-in enabled (`optin: "double"`), confirmation emails are sent to subscribers. The `confirmation_from` field allows you to specify a custom sender address for these emails. This is useful when running multiple services through a single listmonk instance, where each service needs to send confirmation emails from its own address.

- If `confirmation_from` is not set, confirmation emails use the system default sender address (configured globally).
- If `confirmation_from` is set to a valid email address, confirmation emails for that list will be sent from that address.

**Important:** The custom `confirmation_from` address can only be used when subscribing a user to **a single list**. Attempting to subscribe a user to multiple lists where any of them has a `confirmation_from` configured will return an error. This ensures clarity and prevents ambiguous email sender configurations. To subscribe to multiple lists, either use lists without `confirmation_from` set, or make separate API calls for each list with a custom sender address.

| Method | Endpoint                                        | Description               |
| :----- | :---------------------------------------------- | :------------------------ |
| GET    | [/api/lists](#get-apilists)                     | Retrieve all lists.       |
| GET    | [/api/public/lists](#get-public-apilists)       | Retrieve public lists.    |
| GET    | [/api/lists/{list_id}](#get-apilistslist_id)    | Retrieve a specific list. |
| POST   | [/api/lists](#post-apilists)                    | Create a new list.        |
| PUT    | [/api/lists/{list_id}](#put-apilistslist_id)    | Update a list.            |
| DELETE | [/api/lists/{list_id}](#delete-apilistslist_id) | Delete a list.            |
| DELETE | [/api/lists](#delete-apilists)                  | Delete multiple lists.    |

______________________________________________________________________

#### GET /api/lists

Retrieve lists.

> **Note:** Lists with `status: archived` are hidden from list selectors in campaigns, public subscription forms, and roles by default. They can only be viewed by filtering with `status=archived` or by viewing all lists without a status filter.

##### Parameters

| Name     | Type     | Required | Description                                                                                        |
| :------- | :------- | :------- | :------------------------------------------------------------------------------------------------- |
| query    | string   |          | String for list name search.                                                                       |
| status   | string   |          | Status to filter lists. Options: active, archived. Defaults to showing all lists if not specified. |
| minimal  | boolean  |          | If true, returns lists without subscriber counts (faster). Defaults to false.                      |
| tag      | []string |          | Tags to filter lists. Repeat in the query for multiple values.                                     |
| order_by | string   |          | Sort field. Options: name, status, created_at, updated_at.                                         |
| order    | string   |          | Sorting order. Options: ASC, DESC.                                                                 |
| page     | number   |          | Page number for pagination.                                                                        |
| per_page | number   |          | Results per page. Set to 'all' to return all results.                                              |

##### Example Request

```shell
# Get all lists
curl -u "api_user:token" -X GET 'http://localhost:9000/api/lists?page=1&per_page=100'

# Get only active lists
curl -u "api_user:token" -X GET 'http://localhost:9000/api/lists?status=active&per_page=100'

# Get archived lists with minimal data
curl -u "api_user:token" -X GET 'http://localhost:9000/api/lists?status=archived&minimal=true&per_page=all'
```

##### Example Response

```json
{
    "data": {
        "results": [
            {
                "id": 1,
                "created_at": "2020-02-10T23:07:16.194843+01:00",
                "updated_at": "2020-03-06T22:32:01.118327+01:00",
                "uuid": "ce13e971-c2ed-4069-bd0c-240e9a9f56f9",
                "name": "Default list",
                "type": "public",
                "optin": "double",
                "status": "active",
                "tags": [
                    "test"
                ],
                "confirmation_from": "noreply@example.com",
                "subscriber_count": 2
            },
            {
                "id": 2,
                "created_at": "2020-03-04T21:12:09.555013+01:00",
                "updated_at": "2020-03-06T22:34:46.405031+01:00",
                "uuid": "f20a2308-dfb5-4420-a56d-ecf0618a102d",
                "name": "get",
                "type": "private",
                "optin": "single",
                "status": "active",
                "tags": [],
                "confirmation_from": null,
                "subscriber_count": 0
            }
        ],
        "total": 5,
        "per_page": 20,
        "page": 1
    }
}
```

______________________________________________________________________

#### GET /api/public/lists

Retrieve public lists with name and uuid to submit a subscription. This is an unauthenticated call to enable scripting to subscription form.

> **Note:** This endpoint only returns lists with `type: public` and `status: active`. Archived lists are never shown on public subscription forms.

##### Example Request

```shell
curl -X GET 'http://localhost:9000/api/public/lists'
```

##### Example Response

```json
[
  {
    "uuid": "55e243af-80c6-4169-8d7f-bc571e0269e9",
    "name": "Opt-in list"
  }
]
```
______________________________________________________________________

#### GET /api/lists/{list_id}

Retrieve a specific list.

##### Parameters

| Name    | Type   | Required | Description                 |
| :------ | :----- | :------- | :-------------------------- |
| list_id | number | Yes      | ID of the list to retrieve. |

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/lists/5'
```

##### Example Response

```json
{
    "data": {
        "id": 5,
        "created_at": "2020-03-07T06:31:06.072483+01:00",
        "updated_at": "2020-03-07T06:31:06.072483+01:00",
        "uuid": "1bb246ab-7417-4cef-bddc-8fc8fc941d3a",
        "name": "Test list",
        "type": "public",
        "optin": "double",
        "status": "active",
        "tags": [],
        "confirmation_from": null,
        "subscriber_count": 0
    }
}
```

______________________________________________________________________

#### POST /api/lists

Create a new list.

##### Parameters

| Name              | Type       | Required | Description                                                                      |
| :---------------- | :--------- | :------- | :------------------------------------------------------------------------------- |
| name              | string     | Yes      | Name of the new list.                                                           |
| type              | string     | Yes      | Type of list. Options: private, public.                                         |
| optin             | string     | Yes      | Opt-in type. Options: single, double.                                           |
| status            | string     | No       | Status of the list. Options: active, archived. Defaults to active.              |
| tags              | string\[\] |          | Associated tags for a list.                                                     |
| description       | string     | No       | Description of the new list.                                                    |
| confirmation_from | string     | No       | Email address to use as the "from" address for confirmation emails (double opt-in). If not set, the system default is used. |

##### Example Request

```shell
curl -u "api_user:token" -X POST 'http://localhost:9000/api/lists' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Test list",
    "type": "public",
    "optin": "double",
    "description": "This is a test list",
    "confirmation_from": "noreply@example.com"
  }'
```

##### Example Response

```json
{
    "data": {
        "id": 5,
        "created_at": "2020-03-07T06:31:06.072483+01:00",
        "updated_at": "2020-03-07T06:31:06.072483+01:00",
        "uuid": "1bb246ab-7417-4cef-bddc-8fc8fc941d3a",
        "name": "Test list",
        "type": "public",
        "optin": "double",
        "status": "active",
        "tags": [],
        "confirmation_from": "noreply@example.com",
        "subscriber_count": 0,
        "description": "This is a test list"
    }
}
```

______________________________________________________________________

#### PUT /api/lists/{list_id}

Update a list.

##### Parameters

| Name              | Type       | Required | Description                                                                      |
| :---------------- | :--------- | :------- | :------------------------------------------------------------------------------- |
| list_id           | number     | Yes      | ID of the list to update.                                                       |
| name              | string     |          | New name for the list.                                                          |
| type              | string     |          | Type of list. Options: private, public.                                         |
| optin             | string     |          | Opt-in type. Options: single, double.                                           |
| status            | string     |          | Status of the list. Options: active, archived.                                  |
| tags              | string\[\] |          | Associated tags for the list.                                                   |
| description       | string     |          | Description of the list.                                                        |
| confirmation_from | string     |          | Email address to use as the "from" address for confirmation emails (double opt-in). If not set, the system default is used. |

##### Example Request

```shell
curl -u "api_user:token" -X PUT 'http://localhost:9000/api/lists/5' \
--form 'name=modified test list' \
--form 'type=private'
```

##### Example Response

```json
{
    "data": {
        "id": 5,
        "created_at": "2020-03-07T06:31:06.072483+01:00",
        "updated_at": "2020-03-07T06:52:15.208075+01:00",
        "uuid": "1bb246ab-7417-4cef-bddc-8fc8fc941d3a",
        "name": "modified test list",
        "type": "private",
        "optin": "single",
        "status": "active",
        "tags": [],
        "confirmation_from": null,
        "subscriber_count": 0,
        "description": "This is a test list"
    }
}
```

______________________________________________________________________

#### DELETE /api/lists/{list_id}

Delete a specific list.

##### Parameters

| Name    | Type   | Required | Description               |
| :------ | :----- | :------- | :------------------------ |
| list_id | Number | Yes      | ID of the list to delete. |

##### Example Request

```shell
curl -u 'api_username:access_token' -X DELETE 'http://localhost:9000/api/lists/1'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### DELETE /api/lists

Delete multiple lists by IDs or by a search query.

> **Note:** Users can only delete lists they have `manage` permission for. Any lists in the query that the user doesn't have permission to manage is ignored.

##### Parameters

| Name  | Type       | Required                      | Description                                                        |
| :---- | :--------- | :---------------------------- | :----------------------------------------------------------------- |
| id    | number\[\] | Yes (if `query` not provided) | One or more list IDs to delete.                                    |
| query | string     | Yes (if `id` not provided)    | Search query to filter lists for deletion (same as the GET query). |

##### Example Request (by IDs)

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/lists?id=10&id=11&id=12'
```

##### Example Request (by search query)

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/lists?query=test%20list'
```

##### Example Response

```json
{
    "data": true
}
```
