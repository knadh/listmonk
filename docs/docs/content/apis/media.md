# API / Media

Method | Endpoint                                             | Description
-------|------------------------------------------------------|---------------------------------
GET    | [/api/media](#get-apimedia)                          | Get uploaded media file
GET    | [/api/media/tags](#get-apimediatags)                  | Get media tags and counts
GET    | [/api/media/{media_id}](#get-apimediamedia_id)       | Get specific uploaded media file
POST   | [/api/media](#post-apimedia)                         | Upload media file
PUT    | [/api/media/{media_id}](#put-apimediamedia_id)          | Update media tags
DELETE | [/api/media/{media_id}](#delete-apimediamedia_id)    | Delete uploaded media file

______________________________________________________________________

#### GET /api/media

Get uploaded media. The optional `query` parameter searches filenames and tags. Repeat the `tag` parameter to filter by exact tags.

##### Example Request

```shell
curl -u "api_user:token" -X GET 'http://localhost:9000/api/media' \
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

#### GET /api/media/tags

Get all tags and item count for each tag for a given provider.

##### Example Request

```shell
curl -u 'api_user:token' 'http://localhost:9000/api/media/tags'
```

##### Example Response

```json
{
    "data": [
        {"tag": "newsletter", "count": 12},
        {"tag": "summer-sale", "count": 4}
    ]
}
```

______________________________________________________________________

#### GET /api/media/{media_id}

Retrieve a specific media.

##### Parameters

| Name          | Type      | Required | Description      |
|:--------------|:----------|:---------|:-----------------|
| media_id      | Number    | Yes      | Media ID.        |

##### Example Request

```shell
curl -u 'api_username:access_token' 'http://localhost:9000/api/media/7' 
```

##### Example Response

```json
{
  "data": 
    {
        "id": 7,
        "uuid": "62e32e97-d6ca-4441-923f-b62607000dd1",
        "filename": "ResumeB.pdf",
        "content_type": "application/pdf",
        "created_at": "2024-08-06T11:28:53.888257+05:30",
        "thumb_url": null,
        "provider": "filesystem",
        "meta": {},
        "url": "http://localhost:9000/uploads/ResumeB.pdf"
    }
}
```
______________________________________________________________________

#### POST /api/media

Upload a media file.

##### Parameters

| Field | Type      | Required | Description         |
|-------|-----------|----------|---------------------|
| file  | File      | Yes      | Media file to upload|
| tags  | String    | No       | Tag to assign. Repeat this multipart field for multiple tags. |

##### Example Request

```shell
curl -u "api_user:token" -X POST 'http://localhost:9000/api/media' \
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

#### PUT /api/media/{media_id}

Update a media item's tags.

```shell
curl -u 'api_user:token' -X PUT 'http://localhost:9000/api/media/1' \
--header 'Content-Type: application/json' \
--data '{"tags": ["newsletter", "summer-sale"]}'
```

Returns the updated media item.

______________________________________________________________________

#### DELETE /api/media/{media_id}

Delete an uploaded media file.

##### Parameters

| Field    | Type      | Required | Description             |
|----------|-----------|----------|-------------------------|
| media_id | number    | Yes      | ID of media file to delete |

##### Example Request

```shell
curl -u "api_user:token" -X DELETE 'http://localhost:9000/api/media/1'
```

##### Example Response

```json
{
    "data": true
}
```
