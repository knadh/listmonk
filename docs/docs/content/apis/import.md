# API / Import

Method   | Endpoint                                        | Description
---------|-------------------------------------------------|------------------------------------------------
GET      | [/api/import/subscribers](#get-apiimportsubscribers) | Retrieve import statistics.
GET      | [/api/import/subscribers/logs](#get-apiimportsubscriberslogs) | Retrieve import logs.
POST     | [/api/import/subscribers](#post-apiimportsubscribers) | Upload a file for bulk subscriber import.
DELETE   | [/api/import/subscribers](#delete-apiimportsubscribers) | Stop and remove an import.

______________________________________________________________________

#### GET /api/import/subscribers

Retrieve the status of an ongoing import.

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/import/subscribers'
```

##### Example Response

```json
{
    "data": {
        "name": "",
        "total": 0,
        "imported": 0,
        "status": "none"
    }
}
```

______________________________________________________________________

#### GET /api/import/subscribers/logs

Retrieve logs from an ongoing import.

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/import/subscribers/logs'
```

##### Example Response

```json
{
    "data": "2020/04/08 21:55:20 processing 'import.csv'\n2020/04/08 21:55:21 imported finished\n"
}
```

______________________________________________________________________

#### POST /api/import/subscribers

Send a CSV (optionally ZIP compressed) file to import subscribers. Use a multipart form POST.

##### Parameters

| Name   | Type        | Required | Description                              |
|:-------|:------------|:---------|:-----------------------------------------|
| params | JSON string | Yes      | Stringified JSON with import parameters. |
| file   | file        | Yes      | File for upload.                         |


#### `params` (JSON string)
| Name      | Type     | Required | Description                                                                                                                        |
|:----------|:---------|:---------|:-----------------------------------------------------------------------------------------------------------------------------------|
| mode      | string   | Yes      | `subscribe` or `blocklist`                                                                                                         |
| delim     | string   | Yes      | Single character indicating delimiter used in the CSV file, eg: `,`                                                                |
| lists     | []number |          | Array of list IDs to subscribe to.                                                                                                 |
| overwrite | bool     |          | Whether to overwrite the subscriber parameters including subscriptions or ignore records that are already present in the database. |

##### Example Request

```shell
curl -u "api_user:token" -X POST 'http://localhost:9000/api/import/subscribers' \
  -F 'params={"mode":"subscribe", "subscription_status":"confirmed", "delim":",", "lists":[1, 2], "overwrite": true}' \
  -F "file=@/path/to/subs.csv"
```

##### Example Response

```json
    {
        "mode": "subscribe", // subscribe or blocklist
        "delim": ",",        // delimiter in the uploaded file
        "lists":[1],         // array of list IDs to import into
        "overwrite": true    // overwrite existing entries or skip them?
    }
```

______________________________________________________________________

#### DELETE /api/import/subscribers

Stop and delete an ongoing import.

##### Example Request

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/import/subscribers' 
```

##### Example Response

```json
{
    "data": {
        "name": "",
        "total": 0,
        "imported": 0,
        "status": "none"
    }
}
```
