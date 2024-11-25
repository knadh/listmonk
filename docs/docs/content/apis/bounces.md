# API / Bounces

Method   | Endpoint                                                | Description
---------|---------------------------------------------------------|------------------------------------------------
GET      | [/api/bounces](#get-apibounces)                         | Retrieve bounce records.
DELETE   | [/api/bounces](#delete-apibounces)                      | Delete all/multiple bounce records.
DELETE   | [/api/bounces/{bounce_id}](#delete-apibouncesbounce_id) | Delete specific bounce record.


______________________________________________________________________

#### GET /api/bounces

Retrieve the bounce records.

##### Parameters

| Name       | Type     | Required | Description                                                      |
|:-----------|:---------|:---------|:-----------------------------------------------------------------|
| campaign_id| number   |          | Bounce record retrieval for particular campaign id               |
| page       | number   |          | Page number for pagination.                                      |
| per_page   | number   |          | Results per page. Set to 'all' to return all results.            |
| source     | string   |          |                                |
| order_by   | string   |          | Fields by which bounce records are ordered. Options:"email", "campaign_name", "source", "created_at".        |
| order      | number   |          | Sorts the result. Allowed values: 'asc','desc'                   |

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/bounces?campaign_id=1&page=1&per_page=2' \ 
    -H 'accept: application/json' -H 'Content-Type: application/x-www-form-urlencoded' \
    --data '{"source":"demo","order_by":"created_at","order":"asc"}'
```

##### Example Response

```json
{
  "data": {
    "results": [
      {
        "id": 839971,
        "type": "hard",
        "source": "demo",
        "meta": {
          "some": "parameter"
        },
        "created_at": "2024-08-20T23:54:22.851858Z",
        "email": "gilles.deleuze@example.app",
        "subscriber_uuid": "32ca1f3e-1a1d-42e1-af04-df0757f420f3",
        "subscriber_id": 60,
        "campaign": {
          "id": 1,
          "name": "Test campaign"
        }
      },
      {
        "id": 839725,
        "type": "hard",
        "source": "demo",
        "meta": {
          "some": "parameter"
        },
        "created_at": "2024-08-20T22:46:36.393547Z",
        "email": "gottfried.leibniz@example.app",
        "subscriber_uuid": "5911d3f4-2346-4bfc-aad2-eb319ab0e879",
        "subscriber_id": 13,
        "campaign": {
          "id": 1,
          "name": "Test campaign"
        }
      }
    ],
    "query": "",
    "total": 528,
    "per_page": 2,
    "page": 1
  }
}
```

______________________________________________________________________

#### DELETE /api/bounces

To delete all bounces.

##### Parameters

| Name    | Type      | Required | Description                          |
|:--------|:----------|:---------|:-------------------------------------|
| all     | bool      | Yes      | Bool to confirm deleting all bounces |

##### Example Request

```shell
curl -u 'api_username:access_token' -X DELETE 'http://localhost:9000/api/bounces?all=true'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### DELETE /api/bounces

To delete multiple bounce records.

##### Parameters

| Name    | Type      | Required | Description                          |
|:--------|:----------|:---------|:-------------------------------------|
| id      | number    | Yes      | Id's of bounce records to delete.    |

##### Example Request

```shell
curl -u 'api_username:access_token' -X DELETE 'http://localhost:9000/api/bounces?id=840965&id=840168&id=840879'
```

##### Example Response

```json
{
    "data": true
}
```

______________________________________________________________________

#### DELETE /api/bounces/{bounce_id}

To delete specific bounce id.

##### Example Request

```shell
curl -u 'api_username:access_token' -X DELETE 'http://localhost:9000/api/bounces/840965'
```

##### Example Response

```json
{
    "data": true
}
```