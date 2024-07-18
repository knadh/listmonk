# API / Templates

| Method | Endpoint                                                                      | Description                    |
|:-------|:------------------------------------------------------------------------------|:-------------------------------|
| GET    | [/api/templates](#get-apitemplates)                                           | Retrieve all templates         |
| GET    | [/api/templates/{template_id}](#get-apitemplates-template_id)                 | Retrieve a template            |
| GET    | [/api/templates/{template_id}/preview](#get-apitemplates-template_id-preview) | Retrieve template HTML preview |
| POST   | [/api/templates](#post-apitemplates)                                          | Create a template              |
| POST   | /api/templates/preview                                                        | Render and preview a template  |
| PUT    | [/api/templates/{template_id}](#put-apitemplatestemplate_id)                  | Update a template              |
| PUT    | [/api/templates/{template_id}/default](#put-apitemplates-template_id-default) | Set default template           |
| DELETE | [/api/templates/{template_id}](#delete-apitemplates-template_id)              | Delete a template              |

______________________________________________________________________

#### GET /api/templates

Retrieve all templates.

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/templates'
```

##### Example Response

```json
{
    "data": [
        {
            "id": 1,
            "created_at": "2020-03-14T17:36:41.288578+01:00",
            "updated_at": "2020-03-14T17:36:41.288578+01:00",
            "name": "Default template",
            "body": "{{ template \"content\" . }}",
            "type": "campaign",
            "is_default": true
        }
    ]
}
```

______________________________________________________________________

#### GET /api/templates/{template_id}

Retrieve a specific template.

##### Parameters

| Name        | Type      | Required | Description                    |
|:------------|:----------|:---------|:-------------------------------|
| template_id | number    | Yes      | ID of the template to retrieve |

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/templates/1'
```

##### Example Response

```json
{
    "data": {
        "id": 1,
        "created_at": "2020-03-14T17:36:41.288578+01:00",
        "updated_at": "2020-03-14T17:36:41.288578+01:00",
        "name": "Default template",
        "body": "{{ template \"content\" . }}",
        "type": "campaign",
        "is_default": true
    }
}
```

______________________________________________________________________

#### GET /api/templates/{template_id}/preview

Retrieve the HTML preview of a template.

##### Parameters

| Name        | Type      | Required | Description                   |
|:------------|:----------|:---------|:------------------------------|
| template_id | number    | Yes      | ID of the template to preview |

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/templates/1/preview'
```

##### Example Response

```html
<p>Hi there</p>
<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis et elit ac elit sollicitudin condimentum non a magna.
	Sed tempor mauris in facilisis vehicula. Aenean nisl urna, accumsan ac tincidunt vitae, interdum cursus massa.
	Interdum et malesuada fames ac ante ipsum primis in faucibus. Aliquam varius turpis et turpis lacinia placerat.
	Aenean id ligula a orci lacinia blandit at eu felis. Phasellus vel lobortis lacus. Suspendisse leo elit, luctus sed
	erat ut, venenatis fermentum ipsum. Donec bibendum neque quis.</p>

<h3>Sub heading</h3>
<p>Nam luctus dui non placerat mattis. Morbi non accumsan orci, vel interdum urna. Duis faucibus id nunc ut euismod.
	Curabitur et eros id erat feugiat fringilla in eget neque. Aliquam accumsan cursus eros sed faucibus.</p>

<p>Here is a link to <a href="https://listmonk.app" target="_blank">listmonk</a>.</p>
```

______________________________________________________________________

#### POST /api/templates

Create a template.

##### Parameters

| Name    | Type      | Required | Description                                   |
|:--------|:----------|:---------|:----------------------------------------------|
| name    | string    | Yes      | Name of the template                          |
| type    | string    | Yes      | Type of the template (`campaign` or `tx`)     |
| subject | string    |          | Subject line for the template (only for `tx`) |
| body    | string    | Yes      | HTML body of the template                     |

##### Example Request

```shell
curl -u "username:password" -X POST 'http://localhost:9000/api/templates' \
-H 'Content-Type: application/json' \
-d '{
    "name": "New template",
    "type": "campaign",
    "subject": "Your Weekly Newsletter",
    "body": "<h1>Header</h1><p>Content goes here</p>"
}'
```

##### Example Response

```json
{
    "data": [
        {
            "id": 1,
            "created_at": "2020-03-14T17:36:41.288578+01:00",
            "updated_at": "2020-03-14T17:36:41.288578+01:00",
            "name": "Default template",
            "body": "{{ template \"content\" . }}",
            "type": "campaign",
            "is_default": true
        }
    ]
}
```

______________________________________________________________________

#### PUT /api/templates/{template_id}

Update a template.

> Refer to parameters from [POST /api/templates](#post-apitemplates)

______________________________________________________________________

#### PUT /api/templates/{template_id}/default

Set a template as the default.

##### Parameters

| Name        | Type      | Required | Description                          |
|:------------|:----------|:---------|:-------------------------------------|
| template_id | number    | Yes      | ID of the template to set as default |

##### Example Request

```shell
curl -u "username:password" -X PUT 'http://localhost:9000/api/templates/1/default'
```

##### Example Response

```json
{
    "data": {
        "id": 1,
        "created_at": "2020-03-14T17:36:41.288578+01:00",
        "updated_at": "2020-03-14T17:36:41.288578+01:00",
        "name": "Default template",
        "body": "{{ template \"content\" . }}",
        "type": "campaign",
        "is_default": true
    }
}
```

______________________________________________________________________

#### DELETE /api/templates/{template_id}

Delete a template.

##### Parameters

| Name        | Type      | Required | Description                  |
|:------------|:----------|:---------|:-----------------------------|
| template_id | number    | Yes      | ID of the template to delete |

##### Example Request

```shell
curl -u "username:password" -X DELETE 'http://localhost:9000/api/templates/35'
```

##### Example Response

```json
{
    "data": true
}
```
