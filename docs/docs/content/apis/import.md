# API / Import

Method     |  Endpoint                                                            |  Description 
-----------|----------------------------------------------------------------------|--------------
`GET`      | [api/import/subscribers](#get-apiimportsubscribers)                  | Gets a import statistics. 
`GET`      | [api/import/subscribers/logs](#get-apiimportsubscriberslogs)         | Get a import statistics .
`POST`     | [api/import/subscribers](#post-apiimportsubscribers)                 | Upload a ZIP file or CSV file to bulk import subscribers. 
`DELETE`   | [api/import/subscribers](#delete-apiimportsubscribers)               | Stops and deletes a import.


#### **`GET`** api/import/subscribers
Gets import status.

##### Example Request 
```shell 
curl -u "username:username" -X GET 'http://localhost:9000/api/import/subscribers'
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

#### **`GET`** api/import/subscribers/logs
Gets import logs.

##### Example Request 
```shell
curl -u "username:username" -X GET 'http://localhost:9000/api/import/subscribers/logs'
```

##### Example Response
```json
{
    "data": "2020/04/08 21:55:20 processing 'import.csv'\n2020/04/08 21:55:21 imported finished\n"
}
```



#### **`POST`** api/import/subscribers
Post a CSV (optionally zipped) file to do a bulk import. The request should be a multipart form POST.


##### Parameters

Name     | Parameter type | Data type       | Required/Optional |  Description
---------|----------------|----------------|-------------------|-----------------------
`params` | Request body | String         | Required          | Stringified JSON with import params
`file` | Request body | File         | Required          | File to upload

***params*** (JSON string)

```json
    {
        "mode": "subscribe", // subscribe or blocklist
        "delim": ",",        // delimiter in the uploaded file
        "lists":[1],         // array of list IDs to import into
        "overwrite": true    // overwrite existing entries or skip them?
    }
```


#### **`DELETE`** api/import/subscribers
Stops and deletes an import.

##### Example Request
```shell
curl -u "username:username" -X DELETE 'http://localhost:9000/api/import/subscribers' 
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