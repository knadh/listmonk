# API / Subscribers

Method   | Endpoint                                                              | Description
---------|-----------------------------------------------------------------------|-----------------------------------------------------------
`GET`    | [/api/subscribers](#get-apisubscribers)                               | Gets all subscribers.
`GET`    | [/api/subscribers/:`id`](#get-apisubscribersid)                       | Gets a single subscriber.
`GET`    | /api/subscribers/lists/:`id`                                          | Gets subscribers in a list.
`GET`    | [/api/subscribers](#get-apisubscriberslist_id)                        | Gets subscribers in one or more lists.
`GET`    | [/api/subscribers](#get-apisubscribers_1)                             | Gets subscribers filtered by an arbitrary SQL expression.
`POST`   | [/api/subscribers](#post-apisubscribers)                              | Creates a new subscriber.
`POST`   | [/api/subscribers](#post-apisubscriberspublic)                        | Unauthenticated API that enables public subscription.
`PUT`    | [/api/subscribers/lists](#put-apisubscriberslists)                    | Modify subscribers' list memberships.
`PUT`    | [/api/subscribers/:`id`](#put-apisubscribersid)                       | Updates a subscriber by ID.
`PUT`    | [/api/subscribers/:`id`/blocklist](#put-apisubscribersidblocklist)    | Blocklists a single subscriber.
`PUT`    | /api/subscribers/blocklist                                            | Blocklists one or more subscribers.
`PUT`    | [/api/subscribers/query/blocklist](#put-apisubscribersqueryblocklist) | Blocklists subscribers with an arbitrary SQL expression.
`DELETE` | [/api/subscribers/:`id`](#delete-apisubscribersid)                    | Deletes a single subscriber.
`DELETE` | [/api/subscribers](#delete-apisubscribers)                            | Deletes one or more subscribers .
`POST`   | [/api/subscribers/query/delete](#post-apisubscribersquerydelete)      | Deletes subscribers with an arbitrary SQL expression.


#### **`GET`** /api/subscribers
Gets all subscribers. 

##### Example Request
```shell
curl -u 'username:password' 'http://localhost:9000/api/subscribers?page=1&per_page=100' 
```

To skip pagination and retrieve all records, pass `per_page=all`.

##### Example Response
```json
{
    "data": {
        "results": [
            {
                "id": 1,
                "created_at": "2020-02-10T23:07:16.199433+01:00",
                "updated_at": "2020-02-10T23:07:16.199433+01:00",
                "uuid": "ea06b2e7-4b08-4697-bcfc-2a5c6dde8f1c",
                "email": "john@example.com",
                "name": "John Doe",
                "attribs": {
                    "city": "Bengaluru",
                    "good": true,
                    "type": "known"
                },
                "status": "enabled",
                "lists": [
                    {
                        "subscription_status": "unconfirmed",
                        "id": 1,
                        "uuid": "ce13e971-c2ed-4069-bd0c-240e9a9f56f9",
                        "name": "Default list",
                        "type": "public",
                        "tags": [
                            "test"
                        ],
                        "created_at": "2020-02-10T23:07:16.194843+01:00",
                        "updated_at": "2020-02-10T23:07:16.194843+01:00"
                    }
                ]
            },
            {
                "id": 2,
                "created_at": "2020-02-18T21:10:17.218979+01:00",
                "updated_at": "2020-02-18T21:10:17.218979+01:00",
                "uuid": "ccf66172-f87f-4509-b7af-e8716f739860",
                "email": "quadri@example.com",
                "name": "quadri",
                "attribs": {},
                "status": "enabled",
                "lists": [
                    {
                        "subscription_status": "unconfirmed",
                        "id": 1,
                        "uuid": "ce13e971-c2ed-4069-bd0c-240e9a9f56f9",
                        "name": "Default list",
                        "type": "public",
                        "tags": [
                            "test"
                        ],
                        "created_at": "2020-02-10T23:07:16.194843+01:00",
                        "updated_at": "2020-02-10T23:07:16.194843+01:00"
                    }
                ]
            },
            {
                "id": 3,
                "created_at": "2020-02-19T19:10:49.36636+01:00",
                "updated_at": "2020-02-19T19:10:49.36636+01:00",
                "uuid": "5d940585-3cc8-4add-b9c5-76efba3c6edd",
                "email": "sugar@example.com",
                "name": "sugar",
                "attribs": {},
                "status": "enabled",
                "lists": []
            }
        ],
        "query": "",
        "total": 3,
        "per_page": 20,
        "page": 1
    }
}
```


#### **`GET`** /api/subscribers/:`id`
 Gets a single subscriber. 

##### Parameters

Name     | Parameter type |Data type       | Required/Optional |  Description
---------|----------------|----------------|-------------------|-----------------------
`id`     | Path parameter | Number         | Required          | The id value of the subscriber you want to get.

##### Example Request
```shell
curl -u 'username:password' 'http://localhost:9000/api/subscribers/1' 
```

##### Example Response
```json
{
    "data": {
        "id": 1,
        "created_at": "2020-02-10T23:07:16.199433+01:00",
        "updated_at": "2020-02-10T23:07:16.199433+01:00",
        "uuid": "ea06b2e7-4b08-4697-bcfc-2a5c6dde8f1c",
        "email": "john@example.com",
        "name": "John Doe",
        "attribs": {
            "city": "Bengaluru",
            "good": true,
            "type": "known"
        },
        "status": "enabled",
        "lists": [
            {
                "subscription_status": "unconfirmed",
                "id": 1,
                "uuid": "ce13e971-c2ed-4069-bd0c-240e9a9f56f9",
                "name": "Default list",
                "type": "public",
                "tags": [
                    "test"
                ],
                "created_at": "2020-02-10T23:07:16.194843+01:00",
                "updated_at": "2020-02-10T23:07:16.194843+01:00"
            }
        ]
    }
}
```



#### **`GET`** /api/subscribers
Gets subscribers in one or more lists. 

##### Parameters

Name      | Parameter type  | Data type   | Required/Optional   | Description
----------|-----------------|-------------|---------------------|---------------
`List_id` | Request body    | Number      | Required            |  ID of the list to fetch subscribers from.

##### Example Request
```shell
curl -u 'username:password' 'http://localhost:9000/api/subscribers?list_id=1&list_id=2&page=1&per_page=100'
```

To skip pagination and retrieve all records, pass `per_page=all`.

##### Example Response

```json
{
  "data": {
    "results": [
      {
        "id": 1,
        "created_at": "2019-06-26T16:51:54.37065+05:30",
        "updated_at": "2019-07-03T11:53:53.839692+05:30",
        "uuid": "5e91dda1-1c16-467d-9bf9-2a21bf22ae21",
        "email": "test@test.com",
        "name": "Test Subscriber",
        "attribs": {
          "city": "Bengaluru",
          "projects": 3,
          "stack": {
            "languages": ["go", "python"]
          }
        },
        "status": "enabled",
        "lists": [
          {
            "subscription_status": "unconfirmed",
            "id": 1,
            "uuid": "41badaf2-7905-4116-8eac-e8817c6613e4",
            "name": "Default list",
            "type": "public",
            "tags": ["test"],
            "created_at": "2019-06-26T16:51:54.367719+05:30",
            "updated_at": "2019-06-26T16:51:54.367719+05:30"
          }
        ]
      }
    ],
    "query": "",
    "total": 1,
    "per_page": 20,
    "page": 1
  }
}
```

#### **`GET`** /api/subscribers
Gets subscribers with an SQL expression.

##### Example Request
```shell
curl -u 'username:password' -X GET 'http://localhost:9000/api/subscribers' \
    --url-query 'page=1' \
    --url-query 'per_page=100' \
    --url-query "query=subscribers.name LIKE 'Test%' AND subscribers.attribs->>'city' = 'Bengaluru'"
```

To skip pagination and retrieve all records, pass `per_page=all`.


>Refer to the [querying and segmentation](/docs/querying-and-segmentation#querying-and-segmenting-subscribers) section for more information on how to query subscribers with SQL expressions.

##### Example Response 
```json
{
  "data": {
    "results": [
      {
        "id": 1,
        "created_at": "2019-06-26T16:51:54.37065+05:30",
        "updated_at": "2019-07-03T11:53:53.839692+05:30",
        "uuid": "5e91dda1-1c16-467d-9bf9-2a21bf22ae21",
        "email": "test@test.com",
        "name": "Test Subscriber",
        "attribs": {
          "city": "Bengaluru",
          "projects": 3,
          "stack": {
            "frameworks": ["echo", "go"],
            "languages": ["go", "python"]
          }
        },
        "status": "enabled",
        "lists": [
          {
            "subscription_status": "unconfirmed",
            "id": 1,
            "uuid": "41badaf2-7905-4116-8eac-e8817c6613e4",
            "name": "Default list",
            "type": "public",
            "tags": ["test"],
            "created_at": "2019-06-26T16:51:54.367719+05:30",
            "updated_at": "2019-06-26T16:51:54.367719+05:30"
          }
        ]
      }
    ],
    "query": "subscribers.name LIKE 'Test%' AND subscribers.attribs-\u003e\u003e'city' = 'Bengaluru'",
    "total": 1,
    "per_page": 20,
    "page": 1
  }
}
```


#### **`POST`** /api/subscribers

Creates a new subscriber.

##### Parameters 

Name                     | Parameter type   | Data type  | Required/Optional | Description
-------------------------|------------------|------------|-------------------|----------------------------
email                    | Request body     | String     | Required          | The email address of the new subscriber.
name                     | Request body     | String     | Required          | The name of the new subscriber. 
status                   | Request body     | String     | Required          | The status of the new subscriber. Can be enabled, disabled or blocklisted. 
lists                    | Request body     | Numbers    | Optional          | Array of list IDs to subscribe to (marked as `unconfirmed` by default).
attribs                  | Request body     | json       | Optional          | JSON list containing new subscriber's attributes.
preconfirm_subscriptions | Request body     | Bool       | Optional          | If `true`, marks subscriptions as `confirmed` and no-optin e-mails are sent for double opt-in lists.

##### Example Request
```shell
curl -u 'username:password' 'http://localhost:9000/api/subscribers' -H 'Content-Type: application/json' \
    --data '{"email":"subsriber@domain.com","name":"The Subscriber","status":"enabled","lists":[1],"attribs":{"city":"Bengaluru","projects":3,"stack":{"languages":["go","python"]}}}'
```

##### Example Response

```json
{
  "data": {
    "id": 3,
    "created_at": "2019-07-03T12:17:29.735507+05:30",
    "updated_at": "2019-07-03T12:17:29.735507+05:30",
    "uuid": "eb420c55-4cfb-4972-92ba-c93c34ba475d",
    "email": "subsriber@domain.com",
    "name": "The Subscriber",
    "attribs": {
      "city": "Bengaluru",
      "projects": 3,
      "stack": { "languages": ["go", "python"] }
    },
    "status": "enabled",
    "lists": [1]
  }
}
```


#### **`POST`** /api/public/subscription

This is a public, unauthenticated API meant for directly integrating forms for public subscription. The API supports both
form encoded or a JSON encoded body. 

##### Parameters 

Name                     | Parameter type   | Data type  | Required/Optional | Description
-------------------------|------------------|------------|-------------------|----------------------------
email                    | Request body     | String     | Required          | The email address of the subscriber.
name                     | Request body     | String     | Optional          | The name of the new subscriber. 
list_uuids               | Request body     | Strings    | Required          | Array of list UUIDs.

##### Example JSON Request
```shell
curl -u 'http://localhost:9000/api/public/subscription' -H 'Content-Type: application/json' \
    --data '{"email":"subsriber@domain.com","name":"The Subscriber", "lists": ["eb420c55-4cfb-4972-92ba-c93c34ba475d", "0c554cfb-eb42-4972-92ba-c93c34ba475d"]}'
```

##### Example Form Request
```shell
curl -u 'http://localhost:9000/api/public/subscription' \
    -d 'email=subsriber@domain.com' -d 'name=The Subscriber' -d 'l=eb420c55-4cfb-4972-92ba-c93c34ba475d' -d 'l=0c554cfb-eb42-4972-92ba-c93c34ba475d'
```

Notice that in form request, there param is `l` that is repeated for multiple lists, and not `lists` like in JSON.

##### Example Response

```json
{
  "data": true
}
```



#### **`PUT`** /api/subscribers/lists

Modify subscribers list memberships.

##### Parameters

Name              | Parameter type | Data type | Required/Optional  | Description
------------------|----------------|-----------|--------------------|-------------------------------------------------------
`ids`             | Request body   | Numbers   | Required           | The ids of the subscribers to be modified.
`action`          | Request body   | String    | Required           | Whether to `add`, `remove`, or `unsubscribe` the users.
`target_list_ids` | Request body   | Numbers   | Required           | The ids of the lists to be modified.
`status`          | Request body   | String    | Required for `add` | `confirmed`, `unconfirmed`, or `unsubscribed` status.

##### Example Request

To subscribe users 1, 2, and 3 to lists 4, 5, and 6:

```shell
curl -u 'username:password' -X PUT 'http://localhost:9000/api/subscribers/lists' \
--data-raw '{"ids": [1, 2, 3], "action": "add", "target_list_ids": [4, 5, 6], "status": "confirmed"}'
```

##### Example Response

``` json
{
    "data": true
} 
```

#### **`PUT`** /api/subscribers/:`id`

Updates a single subscriber.

##### Parameters 

Parameters are the same as [POST /api/subscribers](#post-apisubscribers) used for subscriber creation. 

> Please note that this is a `PUT` request, so all the parameters have to be set. For example if you don't provide `lists`, the subscriber will be deleted from all the lists he was previously signed on.

#### **`PUT`** /api/subscribers/:`id`/blocklist
Blocklists a single subscriber.

##### Parameters 

Name  | Parameter type | Data type  | Required/Optional | Description 
------|----------------|------------|-------------------|-------------
`id`  | Path parameter | Number     | Required          | The id value of the subscriber you want to blocklist.

##### Example Request 

```shell
curl -u 'username:password' -X PUT 'http://localhost:9000/api/subscribers/9/blocklist'
```

##### Example Response 

``` json
{
    "data": true
} 
```

#### **`PUT`** /api/subscribers/query/blocklist 
Blocklists subscribers with an arbitrary sql expression.

##### Example Request
``` shell
curl -u 'username:password' -X PUT 'http://localhost:9000/api/subscribers/query/blocklist' \
--data-raw '"query=subscribers.name LIKE '\''John Doe'\'' AND subscribers.attribs->>'\''city'\'' = '\''Bengaluru'\''"'
```

>Refer to the [querying and segmentation](/querying-and-segmentation#querying-and-segmenting-subscribers) section for more information on how to query subscribers with SQL expressions.


##### Example Response

``` json
{
    "data": true
}
```

#### **`DELETE`** /api/subscribers/:`id`
Deletes a single subscriber. 

##### Parameters 

Name    | Parameter type   | Data type   | Required/Optional  |  Description
--------|------------------|-------------|--------------------|------------------
`id`    | Path parameter   | Number      | Required           | The id of the subscriber you want to delete.

##### Example  Request 

``` shell
curl -u 'username:password' -X DELETE 'http://localhost:9000/api/subscribers/9'
```

##### Example Response 

``` json
{
    "data": true
}
```

#### **`DELETE`** /api/subscribers 
Deletes one or more subscribers.

##### Parameters 

Name    |   Parameter type    | Data type      |   Required/Optional   | Description
--------|---------------------|----------------|-----------------------|--------------
id      | Query parameters    | Number         |  Required             | The id of the subscribers you want to delete.

##### Example Request

``` shell
curl -u 'username:password' -X DELETE 'http://localhost:9000/api/subscribers?id=10&id=11'
```

##### Example Response 

``` json 
{
    "data": true
}
```



#### **`POST`** /api/subscribers/query/delete 
Deletes subscribers with an arbitrary SQL expression.

##### Example Request
``` shell
curl -u 'username:password' -X POST 'http://localhost:9000/api/subscribers/query/delete' \
--data-raw '"query=subscribers.name LIKE '\''John Doe'\'' AND subscribers.attribs->>'\''city'\'' = '\''Bengaluru'\''"'
```

>Refer to the [querying and segmentation](/querying-and-segmentation#querying-and-segmenting-subscribers) section for more information on how to query subscribers with SQL expressions.


##### Example Response
``` json
{
    "data": true
}
```
