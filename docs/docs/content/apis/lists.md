# API / Lists

| Method | Endpoint                                        | Description               |
|:-------|:------------------------------------------------|:--------------------------|
| GET    | [/api/lists](#get-apilists)                     | Retrieve all lists.       |
| GET    | [/api/lists/{list_id}](#get-apilistslist_id)    | Retrieve a specific list. |
| POST   | [/api/lists](#post-apilists)                    | Create a new list.        |
| PUT    | [/api/lists/{list_id}](#put-apilistslist_id)    | Update a list.            |
| DELETE | [/api/lists/{list_id}](#delete-apilistslist_id) | Delete a list.            |

______________________________________________________________________

#### GET /api/lists

Retrieve lists.

##### Parameters

| Name     | Type     | Required | Description                                                      |
|:---------|:---------|:---------|:-----------------------------------------------------------------|
| query    | string   |          | string for list name search.                                     |
| status   | []string |          | Status to filter lists. Repeat in the query for multiple values. |
| tags     | []string |          | Tags to filter lists. Repeat in the query for multiple values.   |
| order_by | string   |          | Sort field. Options: name, status, created_at, updated_at.       |
| order    | string   |          | Sorting order. Options: ASC, DESC.                               |
| page     | number   |          | Page number for pagination.                                      |
| per_page | number   |          | Results per page. Set to 'all' to return all results.            |

##### Example Request

```shell
curl -u "username:username" -X GET 'http://localhost:9000/api/lists?page=1&per_page=100'
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
                "tags": [
                    "test"
                ],
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
                "tags": [],
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

#### GET /api/lists/{list_id}

Retrieve a specific list.

##### Parameters

| Name    | Type      | Required | Description                 |
|:--------|:----------|:---------|:----------------------------|
| list_id | number    | Yes      | ID of the list to retrieve. |

##### Example Request

```shell
curl -u "username:username" -X GET 'http://localhost:9000/api/lists/5'
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
        "tags": [],
        "subscriber_count": 0
    }
}
```

______________________________________________________________________

#### POST /api/lists

Create a new list.

##### Parameters

| Name  | Type      | Required | Description                             |
|:------|:----------|:---------|:----------------------------------------|
| name  | string    | Yes      | Name of the new list.                   |
| type  | string    | Yes      | Type of list. Options: private, public. |
| optin | string    | Yes      | Opt-in type. Options: single, double.   |
| tags  | string\[\]  |          | Associated tags for a list.             |

##### Example Request

```shell
curl -u "username:username" -X POST 'http://localhost:9000/api/lists'
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
        "tags": [],
        "subscriber_count": 0
    }
}
null
```

______________________________________________________________________

#### PUT /api/lists/{list_id}

Update a list.

##### Parameters

| Name    | Type      | Required | Description                             |
|:--------|:----------|:---------|:----------------------------------------|
| list_id | number    | Yes      | ID of the list to update.               |
| name    | string    |          | New name for the list.                  |
| type    | string    |          | Type of list. Options: private, public. |
| optin   | string    |          | Opt-in type. Options: single, double.   |
| tags    | string\[\]  |          | Associated tags for the list.           |

##### Example Request

```shell
curl -u "username:username" -X PUT 'http://localhost:9000/api/lists/5' \
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
        "tags": [],
        "subscriber_count": 0
    }
}
```

______________________________________________________________________

#### DELETE /api/lists/{list_id}

Delete a specific subscriber.

##### Parameters

| Name    | Type      | Required | Description               |
|:--------|:----------|:---------|:--------------------------|
| list_id | Number    | Yes      | ID of the list to delete. |

##### Example Request

```shell
curl -u 'username:password' -X DELETE 'http://localhost:9000/api/lists/1'
```

##### Example Response

```json
{
    "data": true
}
```
