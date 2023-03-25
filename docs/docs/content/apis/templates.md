# API / Templates

Method               |        Endpoint                         |  Description
---------------------|-----------------------------------------|-----------------------------------------------------
`GET`           | [/api/templates](#get-apitemplates)                               | Gets all templates.
`GET`            | [/api/templates/:`template_id`](#get-apitemplatestemplate_id)                | Gets a single template.
`GET`            | [/api/templates/:`template_id`/preview](#get-apitemplatestemplate_idpreview)         | Gets the HTML preview of a template.
`POST`           | /api/templates/preview                      |     
`POST`          | /api/templates                               | Creates a template.
`PUT`            | /api/templates/:`template_id`                 | Modifies a template.
`PUT`     | [/api/templates/:`template_id`/default](#put-apitemplatestemplate_iddefault)        | Sets a template to the default template.
`DELETE`         | [/api/templates/:`template_id`](#delete-apitemplatestemplate_id)     | Deletes a template. 

#### **`GET`** /api/templates
Gets all templates.

##### Example Request
 ```shell
 curl -u "username:username" -X GET 'http://localhost:9000/api/templates'
 ```

##### Example Response
``` json
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

#### **`GET`** /api/templates/:`template_id`
Gets a single template.

##### Parameters
Name        |   Parameter Type       | Data Type           | Required/Optional       | Description
------------|------------------------|---------------------|-------------------------|------------------------------------------
`template_id` | Path Parameter         | Number              |    Required             | The id value of the template you want to get.

##### Example Request
``` shell
curl -u "username:username" -X GET 'http://localhost:9000/api/templates/1'
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

#### **`GET`** /api/templates/:`template_id`/preview
Gets the HTML preview of a template body.

##### Parameters
Name        |  Parameter Type      | Data  Type      | Required/Optional      | Description
------------|----------------------|-----------------|------------------------|---------------------------------
`template_id` | Path Parameter       | Number          | Required               | The id value of the template whose html preview you want to get.

##### Example Request
``` shell
curl -u "username:username" -X GET 'http://localhost:9000/api/templates/1/preview'
```

##### Example Response
``` html
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

#### **`PUT`** /api/templates/:`template_id`/default
Sets a template to the default template.

##### Parameters
Name        |   Parameter Type       | Data Type           | Required/Optional       | Description
------------|------------------------|---------------------|-------------------------|------------------------------------------
`template_id` | Path Parameter         | Number              |    Required             | The id value of the template you want to set to the default template.


##### Example Request
``` shell
curl -u "username:username" -X PUT 'http://localhost:9000/api/templates/1/default'
```

##### Example Response
``` json
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


#### **`DELETE`** /api/templates/:`template_id`
Deletes a template.

##### Parameters
Name        |   Parameter Type       | Data Type           | Required/Optional       | Description
------------|------------------------|---------------------|-------------------------|------------------------------------------
`template_id` | Path Parameter         | Number              |    Required             | The id value of the template you want to delete.


##### Example Request
``` shell
curl -u "username:username" -X DELETE 'http://localhost:9000/api/templates/35'
```

##### Example Response
```json
{
    "data": true
}
```