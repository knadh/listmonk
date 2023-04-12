# API / Media
Method        |         Endpoint                                             |       Description
--------------|--------------------------------------------------------------|----------------------------------------------
`GET`         | [/api/media](#get-apimedia)                                  | Gets an uploaded media file.
`POST`        | [/api/media](#post-apimedia)                                 | Uploads a media file.
`DELETE`      | [/api/media/:`media_id`](#delete-apimediamedia_id)             | Deletes uploaded media files. 

#### **`GET`** /api/media
Gets an uploaded media file.

##### Example Request
```shell
curl -u "username:username" -X GET 'http://localhost:9000/api/media' \
--header 'Content-Type: multipart/form-data; boundary=--------------------------093715978792575906250298'
```

##### Example Response
```json
{
    "data": [
        {
            "id": 1,
            "uuid": "ec7b45ce-1408-4e5c-924e-965326a20287",
            "filename": "Media file",
            "created_at": "2020-04-08T22:43:45.080058+01:00",
            "thumb_uri": "/uploads/image_thumb.jpg",
            "uri": "/uploads/image.jpg"
        }
    ]
}
```

Response definitions
The following table describes each item in the response.

|Response item |Description |Data type |
|:---------------:|:------------|:----------:|
|data|Array of the media file objects, which contains an information about the uploaded media files|array|
|id|Media file object ID|number (int)|
|uuid|Media file uuuid|string (uuid)|
|filename|Name of the media file|string|
|created_at|Date and time, when the media file object was created|String (localDateTime)|
|thumb_uri|The thumbnail URI of the media file|string|
|uri|URI of the media file|string|

#### **`POST`** /api/media
Uploads a media file.

##### Parameters
Nam        |  Parameter Type       |  Data Type        |     Required/Optional   |   Description
-----------|-----------------------|-------------------|-------------------------|---------------------------------
file       |  Request body         |  Media file       |     Required            | The media file to be uploaded.


##### Example Request
```shell 
curl -u "username:username" -X POST 'http://localhost:9000/api/media' \
--header 'Content-Type: multipart/form-data; boundary=--------------------------183679989870526937212428' \
--form 'file=@/path/to/image.jpg'
```

##### Example Response
``` json
{
    "data": {
        "id": 1,
        "uuid": "ec7b45ce-1408-4e5c-924e-965326a20287",
        "filename": "Media file",
        "created_at": "2020-04-08T22:43:45.080058+01:00",
        "thumb_uri": "/uploads/image_thumb.jpg",
        "uri": "/uploads/image.jpg"
    }
}
```
Response definitions

|Response item |Description |Data type |
|:---------------:|:------------:|:----------:|
|data|True means that the media file was successfully uploaded |boolean|

#### **`DELETE`** /api/media/:`media_id`
Deletes an uploaded media file.

##### Parameters
Name            |   Parameter Type        | Data Type          | Required/Optional       | Description
----------------|-------------------------|--------------------|-------------------------|--------------------------
`Media_id`       | Path Parameter          | Number             | Required                | The id of the media file you want to delete.


##### Example Request
```shell
curl -u "username:username" -X DELETE 'http://localhost:9000/api/media/1'
```


##### Example Response
```json
{
    "data": true
}
```

Response definitions

|Response item |Description |Data type |
|:---------------:|:------------:|:----------:|
|data|True means that the media file was successfully deleted |boolean|
