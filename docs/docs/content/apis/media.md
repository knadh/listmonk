# API / Media

Method | Endpoint                                       | Description
-------|------------------------------------------------|------------------------------
GET    | [/api/media](#get-apimedia)                                     | Get uploaded media file
POST   | [/api/media](#post-apimedia)                                     | Upload media file
DELETE | [/api/media/{media_id}](#delete-apimediamedia_id)                          | Delete uploaded media file

______________________________________________________________________

#### GET /api/media

Get an uploaded media file.

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/media' \
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
            "thumb_url": "/uploads/image_thumb.jpg",
            "uri": "/uploads/image.jpg"
        }
    ]
}
```

______________________________________________________________________

#### POST /api/media

Upload a media file.

##### Parameters

| Field | Type      | Required | Description         |
|-------|-----------|----------|---------------------|
| file  | File      | Yes      | Media file to upload|

##### Example Request

```shell
curl -u "username:password" -X POST 'http://localhost:9000/api/media' \
--header 'Content-Type: multipart/form-data; boundary=--------------------------183679989870526937212428' \
--form 'file=@/path/to/image.jpg'
```

##### Example Response

```json
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

______________________________________________________________________

#### DELETE /api/media/{media_id}

Delete an uploaded media file.

##### Parameters

| Field    | Type      | Required | Description             |
|----------|-----------|----------|-------------------------|
| media_id | number    | Yes      | ID of media file to delete |

##### Example Request

```shell
curl -u "username:password" -X DELETE 'http://localhost:9000/api/media/1'
```

##### Example Response

```json
{
    "data": true
}
```
