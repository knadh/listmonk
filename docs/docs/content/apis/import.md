# API / Import

Method   | Endpoint                                        | Description
---------|-------------------------------------------------|------------------------------------------------
GET      | [/api/import/subscribers](#get-apiimportsubscribers) | Retrieve import statistics.
GET      | [/api/import/subscribers/logs](#get-apiimportsubscriberslogs) | Retrieve import logs.
POST     | [/api/import/subscribers](#post-apiimportsubscribers) | Upload a file for bulk subscriber import.
DELETE   | [/api/import/subscribers](#delete-apiimportsubscribers) | Stop and remove an import.

______________________________________________________________________

#### GET /api/import/subscribers

Retrieve the status of an import.

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/import/subscribers'
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

Retrieve logs related to imports.

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/import/subscribers/logs'
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
| file   | File        | Yes      | File for upload.                         |

**`params`** (JSON string)

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
curl -u "username:password" -X DELETE 'http://localhost:9000/api/import/subscribers' 
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
