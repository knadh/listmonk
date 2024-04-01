# API / Subscribers

| Method | Endpoint                                                                                | Description                                    |
| ------ | --------------------------------------------------------------------------------------- | ---------------------------------------------- |
| GET    | [/api/subscribers](#get-apisubscribers)                                                 | Query and retrieve subscribers.                |
| GET    | [/api/subscribers/{subscriber_id}](#get-apisubscriberssubscriber_id)                    | Retrieve a specific subscriber.                |
| POST   | [/api/subscribers](#post-apisubscribers)                                                | Create a new subscriber.                       |
| POST   | [/api/public/subscription](#post-apipublicsubscription)                                 | Create a public subscription.                  |
| PUT    | [/api/subscribers/lists](#put-apisubscriberslists)                                      | Modify subscriber list memberships.            |
| PUT    | [/api/subscribers/{subscriber_id}](#put-apisubscriberssubscriber_id)                    | Update a specific subscriber.                  |
| PUT    | [/api/subscribers/{subscriber_id}/blocklist](#put-apisubscriberssubscriber_idblocklist) | Blocklist a specific subscriber.               |
| PUT    | /api/subscribers/blocklist                                                              | Blocklist one or more subscribers.             |
| PUT    | [/api/subscribers/query/blocklist](#put-apisubscribersqueryblocklist)                   | Blocklist subscribers based on SQL expression. |
| DELETE | [/api/subscribers/{subscriber_id}](#delete-apisubscriberssubscriber_id)                 | Delete a specific subscriber.                  |
| DELETE | [/api/subscribers](#delete-apisubscribers)                                              | Delete one or more subscribers.                |
| POST   | [/api/subscribers/query/delete](#post-apisubscribersquerydelete)                        | Delete subscribers based on SQL expression.    |

______________________________________________________________________

#### GET /api/subscribers

Retrieve all subscribers.

##### Query parameters

| Name                | Type   | Required | Description                                                           |
|:--------------------|:-------|:---------|:----------------------------------------------------------------------|
| query               | string |          | Subscriber search by SQL expression.                                  |
| list_id             | int[]  |          | ID of lists to filter by. Repeat in the query for multiple values.    |
| subscription_status | string |          | Subscription status to filter by if there are one or more `list_id`s. |
| order_by            | string |          | Result sorting field. Options: name, status, created_at, updated_at.  |
| order               | string |          | Sorting order: ASC for ascending, DESC for descending.                |
| page                | number |          | Page number for paginated results.                                    |
| per_page            | number |          | Results per page. Set as 'all' for all results.                       |

##### Example Request

```shell
curl -u 'username:password' 'http://localhost:9000/api/subscribers?page=1&per_page=100' 
```

```shell
curl -u 'username:password' 'http://localhost:9000/api/subscribers?list_id=1&list_id=2&page=1&per_page=100'
```

```shell
curl -u 'username:password' -X GET 'http://localhost:9000/api/subscribers' \
    --url-query 'page=1' \
    --url-query 'per_page=100' \
    --url-query "query=subscribers.name LIKE 'Test%' AND subscribers.attribs->>'city' = 'Bengaluru'"
```

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

______________________________________________________________________

#### GET /api/subscribers/{subscriber_id}

Retrieve a specific subscriber.

##### Parameters

| Name          | Type      | Required | Description      |
|:--------------|:----------|:---------|:-----------------|
| subscriber_id | Number    | Yes      | Subscriber's ID. |

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

______________________________________________________________________


#### POST /api/subscribers

Create a new subscriber.

##### Parameters

| Name                     | Type      | Required | Description                                                                                          |
|:-------------------------|:----------|:---------|:-----------------------------------------------------------------------------------------------------|
| email                    | string    | Yes      | Subscriber's email address.                                                                          |
| name                     | string    | Yes      | Subscriber's name.                                                                                   |
| status                   | string    | Yes      | Subscriber's status: `enabled`, `blocklisted`.                                           |
| lists                    | number\[\]  |          | List of list IDs to subscribe to.                                                                    |
| attribs                  | JSON      |          | Attributes of the new subscriber.                                                                    |
| preconfirm_subscriptions | bool      |          | If true, subscriptions are marked as confirmed and no-optin emails are sent for double opt-in lists. |

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

______________________________________________________________________

#### POST /api/public/subscription

Create a public subscription, accepts both form encoded or JSON encoded body.

##### Parameters

| Name       | Type      | Required | Description                 |
|:-----------|:----------|:---------|:----------------------------|
| email      | string    | Yes      | Subscriber's email address. |
| name       | string    |          | Subscriber's name.          |
| list_uuids | string\[\]  | Yes      | List of list UUIDs.         |

##### Example JSON Request

```shell
curl 'http://localhost:9000/api/public/subscription' -H 'Content-Type: application/json' \
    --data '{"email":"subsriber@domain.com","name":"The Subscriber","list_uuids": ["eb420c55-4cfb-4972-92ba-c93c34ba475d", "0c554cfb-eb42-4972-92ba-c93c34ba475d"]}'
```

##### Example Form Request

```shell
curl -u 'http://localhost:9000/api/public/subscription' \
    -d 'email=subsriber@domain.com' -d 'name=The Subscriber' -d 'l=eb420c55-4cfb-4972-92ba-c93c34ba475d' -d 'l=0c554cfb-eb42-4972-92ba-c93c34ba475d'
```

Note: For form request, use `l` for multiple lists instead of `lists`.

##### Example Response

```json
{
  "data": true
}
```

______________________________________________________________________

#### PUT /api/subscribers/lists

Modify subscriber list memberships.

##### Parameters

| Name            | Type      | Required           | Description                                                       |
|:----------------|:----------|:-------------------|:------------------------------------------------------------------|
| ids             | number\[\]  | Yes                | Array of user IDs to be modified.                                 |
| action          | string    | Yes                | Action to be applied: `add`, `remove`, or `unsubscribe`.          |
| target_list_ids | number\[\]  | Yes                | Array of list IDs to be modified.                                 |
| status          | string    | Required for `add` | Subscriber status: `confirmed`, `unconfirmed`, or `unsubscribed`. |

##### Example Request

```shell
curl -u 'username:password' -X PUT 'http://localhost:9000/api/subscribers/lists' \
-H 'Content-Type: application/json' \
--data-raw '{"ids": [1, 2, 3], "action": "add", "target_list_ids": [4, 5, 6], "status": "confirmed"}'
```

##### Example Response

```json
{
    "data": true
} 
```

______________________________________________________________________

#### PUT /api/subscribers/{subscriber_id}

Update a specific subscriber.

> Refer to parameters from [POST /api/subscribers](#post-apisubscribers). Note: All parameters must be set, if not, the subscriber will be removed from all previously assigned lists.

______________________________________________________________________

#### PUT /api/subscribers/{subscriber_id}/blocklist

Blocklist a specific subscriber.

##### Parameters

| Name          | Type      | Required | Description      |
|:--------------|:----------|:---------|:-----------------|
| subscriber_id | Number    | Yes      | Subscriber's ID. |

##### Example Request

```shell
curl -u 'username:password' -X PUT 'http://localhost:9000/api/subscribers/9/blocklist'
```

##### Example Response

```json
{
    "data": true
} 
```

______________________________________________________________________

#### PUT /api/subscribers/query/blocklist

Blocklist subscribers based on SQL expression.

> Refer to the [querying and segmentation](../querying-and-segmentation.md#querying-and-segmenting-subscribers) section for more information on how to query subscribers with SQL expressions.

##### Example Request

```shell
curl -u 'username:password' -X PUT 'http://localhost:9000/api/subscribers/query/blocklist' \
--data-raw '"query=subscribers.name LIKE '\''John Doe'\'' AND subscribers.attribs->>'\''city'\'' = '\''Bengaluru'\''"'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### DELETE /api/subscribers/{subscriber_id}

Delete a specific subscriber.

##### Parameters

| Name          | Type      | Required | Description      |
|:--------------|:----------|:---------|:-----------------|
| subscriber_id | Number    | Yes      | Subscriber's ID. |

##### Example Request

```shell
curl -u 'username:password' -X DELETE 'http://localhost:9000/api/subscribers/9'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### DELETE /api/subscribers

Delete one or more subscribers.

##### Parameters

| Name | Type      | Required | Description                |
|:-----|:----------|:---------|:---------------------------|
| id   | number\[\]  | Yes      | Array of subscriber's IDs. |

##### Example Request

```shell
curl -u 'username:password' -X DELETE 'http://localhost:9000/api/subscribers?id=10&id=11'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### POST /api/subscribers/query/delete

Delete subscribers based on SQL expression.

##### Example Request

```shell
curl -u 'username:password' -X POST 'http://localhost:9000/api/subscribers/query/delete' \
--data-raw '"query=subscribers.name LIKE '\''John Doe'\'' AND subscribers.attribs->>'\''city'\'' = '\''Bengaluru'\''"'
```

##### Example Response

```json
{
    "data": true
}
```
