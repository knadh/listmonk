# API / Lists
Method      | Endpoint                                             | Description
------------|------------------------------------------------------|----------------------------------------------
`GET`       | [/api/lists](#get-apilists)                          | Gets all lists.
`GET`       | [/api/lists/:`list_id`](#get-apilistslist_id)        | Gets a single list.
`POST`      | [/api/lists](#post-apilists)                         | Creates a new list.
`PUT`       | /api/lists/:`list_id`                                | Modifies a list.
`DELETE`    | [/api/lists/:`list_id`](#put-apilistslist_id)        | Deletes a list.


#### **`GET`** /api/lists
Gets lists.

##### Parameters
Name       | Type   | Required/Optional  | Description
-----------|--------|--------------------|-----------------------------------------
`query`    | string | Optional           | Optional string to search a list by name.
`order_by` | string | Optional           | Field to sort results by. `name|status|created_at|updated_at`
`order`    | string | Optional           | `ASC|DESC`Sort by ascending or descending order.
`page`     | number | Optional           | Page number for paginated results.
`per_page` | number | Optional           | Results to return per page. Setting this to `all` skips pagination and returns all results.

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

#### **`GET`** /api/lists/:`list_id`
Gets a single list.

##### Parameters
Name      | Parameter type     | Data type   | Required/Optional   | Description
----------|--------------------|-------------|---------------------|---------------------
`list_id` | Path parameter     | number      | Required            |  The id value of the list you want to get.

##### Example Request
``` shell
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

#### **`POST`** /api/lists
Creates a new list.

##### Parameters
Name    | Parameter type  | Data type   | Required/Optional  | Description
--------|-----------------|-------------|--------------------|----------------
name    | Request body    | string      | Required           | The new list name.  
type    | Request body    | string      | Required           | List type, can be set to `private` or `public`.
optin   | Request body    | string      | Required           | `single` or `double` optin.
tags    | Request body    | string[]    | Optional           | The tags associated with the list.

##### Example Request
``` shell
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

#### **`PUT`** /api/lists/`list_id`
Modifies a list.

##### Parameters
Name      |  Parameter type    | Data type    | Required/Optional     | Description
----------|--------------------|--------------|-----------------------|-------------------------
`list_id` | Path parameter     | number       | Required              | The id of the list to be modified.
name      | Request body       | string       | Optional              | The name which the old name will be modified to.
type      | Request body       | string       | Optional              | List type, can be set to `private` or `public`.
optin     | Request body       | string       | Optional              | `single` or `double` optin.
tags      | Request body       | string[]     | Optional              | The tags associated with the list.

##### Example Request
```shell
curl -u "username:username" -X PUT 'http://localhost:9000/api/lists/5' \
--form 'name=modified test list' \
--form 'type=private'
```

##### Example Response
``` json
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
