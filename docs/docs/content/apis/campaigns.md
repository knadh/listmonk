# API / Campaigns

Method |   Endpoint                                                                | Description                  
-------|---------------------------------------------------------------------------|-----------------------------
`GET`  | [/api/campaigns](#get-apicampaigns)                                       |  Gets all campaigns.
`GET`  | [/api/campaigns/:`campaign_id`](#get-apicampaignscampaign_id)                |  Gets a single campaign.
`GET`  | [/api/campaigns/:`campaign_id`/preview](#get-apicampaignscampaign_idpreview)  |  Gets the HTML preview of a campaign body.
`GET`  | [/api/campaigns/running/stats](#get-apicampaignsrunningstats)              |  Gets the stats of a given set of campaigns.
`POST` | [/api/campaigns](#post-apicampaigns)                                       |  Creates a new campaign.
`POST` | /api/campaigns/:`campaign_id`/test                         |  Posts campaign message to arbitrary subscribers for testing.
`PUT`  | /api/campaigns/:`campaign_id`                                                |  Modifies a campaign.
`PUT`  | [/api/campaigns/:`campaign_id`/status](#put-apicampaignscampaign_idstatus)   |  Start / pause / cancel / schedule a campaign.
`DELETE`  | [/api/campaigns/:`campaign_id`](#delete-apicampaignscampaign_id)          |  Deletes a campaign. 

#### ```GET``` /api/campaigns

Gets all campaigns.

##### Example Request

```shell
 curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns?page=1&per_page=100'
```

##### Parameters
Name    | Type   | Required/Optional   | Description
--------|--------------------|-------------|---------------------|---------------------
`query` | string      | Optional            |  Optional string to search a list by name.
`order_by` | string      | Optional            |  Field to sort results by. `name|status|created_at|updated_at`
`order` | string      | Optional            |  `ASC|DESC`Sort by ascending or descending order.
`page` | number      | Optional            |  Page number for paginated results.
`per_page` | number      | Optional            |  Results to return per page. Setting this to `all` skips pagination and returns all results.


##### Example Response

``` json
{
    "data": {
        "results": [
            {
                "id": 1,
                "created_at": "2020-03-14T17:36:41.29451+01:00",
                "updated_at": "2020-03-14T17:36:41.29451+01:00",
                "CampaignID": 0,
                "views": 0,
                "clicks": 0,
                "lists": [
                    {
                        "id": 1,
                        "name": "Default list"
                    }
                ],
                "started_at": null,
                "to_send": 0,
                "sent": 0,
                "uuid": "57702beb-6fae-4355-a324-c2fd5b59a549",
                "type": "regular",
                "name": "Test campaign",
                "subject": "Welcome to listmonk",
                "from_email": "No Reply <noreply@yoursite.com>",
                "body": "<h3>Hi {{ .Subscriber.FirstName }}!</h3>\n\t\t\tThis is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.",
                "send_at": "2020-03-15T17:36:41.293233+01:00",
                "status": "draft",
                "content_type": "richtext",
                "tags": [
                    "test-campaign"
                ],
                "template_id": 1,
                "messenger": "email"
            }
        ],
        "query": "",
        "total": 1,
        "per_page": 20,
        "page": 1
    }
}
```

#### ```GET``` /api/campaigns/:`campaign_id`

Gets a single campaign.

##### Parameters 
Name        | Parameter Type    | Data Type    | Required/Optional    | Description
------------|-------------------|--------------|----------------------|-----------------------------
`campaign_id` | Path Parameter    | Number       | Required             | The id  value of the campaign you want to get.


##### Example Request

``` shell
curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns/1'
```

##### Example Response

``` json
{
    "data": {
        "id": 1,
        "created_at": "2020-03-14T17:36:41.29451+01:00",
        "updated_at": "2020-03-14T17:36:41.29451+01:00",
        "CampaignID": 0,
        "views": 0,
        "clicks": 0,
        "lists": [
            {
                "id": 1,
                "name": "Default list"
            }
        ],
        "started_at": null,
        "to_send": 0,
        "sent": 0,
        "uuid": "57702beb-6fae-4355-a324-c2fd5b59a549",
        "type": "regular",
        "name": "Test campaign",
        "subject": "Welcome to listmonk",
        "from_email": "No Reply <noreply@yoursite.com>",
        "body": "<h3>Hi {{ .Subscriber.FirstName }}!</h3>\n\t\t\tThis is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.",
        "send_at": "2020-03-15T17:36:41.293233+01:00",
        "status": "draft",
        "content_type": "richtext",
        "tags": [
            "test-campaign"
        ],
        "template_id": 1,
        "messenger": "email"
    }
}
```


 


#### ```GET``` /api/campaigns/:`campaign_id`/preview 

Gets the html preview of a campaign body.

##### Parameters 

Name        | Parameter Type    | Data Type    | Required/Optional    | Description
------------|-------------------|--------------|----------------------|-----------------------------
`campaign_id` | Path Parameter    | Number       | Required             | The id value of the campaign to be previewed.


##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns/1/preview'
```

##### Example Response

``` html
<h3>Hi John!</h3>
This is a test e-mail campaign. Your second name is Doe and you are from Bengaluru.
```

#### ```GET``` /api/campaigns/running/stats

Gets the running stat of a given set of campaigns.

##### Parameters

Name        | Parameter Type   | Data Type  | Required/Optional   | Description
------------|------------------|------------|---------------------|--------------------------------
campaign_id | Query Parameters | Number     | Required            | The id values of the campaigns whose stat you want to get.


##### Example Request

``` shell
curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns/running/stats?campaign_id=1'
```

##### Example Response

``` json
{
    "data": []
}
```





### ```POST ``` /api/campaigns

Creates a new campaign.

#### Parameters
| Name           | Data type | Required/Optional | Description                                                                                            |
|----------------|-----------|-------------------|--------------------------------------------------------------------------------------------------------|
| `name`         | String    | Required          | Name of the campaign.                                                                                  |
| `subject`      | String    | Required          | (E-mail) subject of the campaign.                                                                      |
| `lists`        | []Number  | Required          | Array of list IDs to send the campaign to.                                                             |
| `from_email`   | String    | Optional          | `From` e-mail to show on the campaign e-mails. If left empty, the default value from settings is used. |
| `type`         | String    | Required          | `regular` or `optin` campaign.                                                                         |
| `content_type` | String    | Required          | `richtext`, `html`, `markdown`, `plain`                                                                |
| `body`         | String    | Required          | Campaign content body.                                                                                 |
| `altbody`      | String    | Optional          | Alternate plain text body for HTML (and richtext) e-mails.                                             |
| `send_at`      | String    | Optional          | A timestamp to schedule the campaign at. Eg: `2021-12-25T06:00:00` (YYYY-MM-DDTHH:MM:SS)               |
| `messenger`    | String    | Optional          | `email` or a custom messenger defined in the settings. If left empty, `email` is used.                 |
| `template_id`  | Number    | Optional          | ID of the template to use. If left empty, the default template is used.                                |
| `tags`         | []String  | Optional          | Array of string tags to mark the campaign.                                                             |




#### Example request

```shell
curl -u "username:password" 'http://localhost:9000/api/campaigns' -X POST -H 'Content-Type: application/json;charset=utf-8' --data-raw '{"name":"Test campaign","subject":"Hello, world","lists":[1],"from_email":"listmonk <noreply@listmonk.yoursite.com>","content_type":"richtext","messenger":"email","type":"regular","tags":["test"],"template_id":1}'
```

#### Example response
```json
{
    "data": {
        "id": 1,
        "created_at": "2021-12-27T11:50:23.333485Z",
        "updated_at": "2021-12-27T11:50:23.333485Z",
        "views": 0,
        "clicks": 0,
        "bounces": 0,
        "lists": [{
            "id": 1,
            "name": "Default list"
        }],
        "started_at": null,
        "to_send": 1,
        "sent": 0,
        "uuid": "90c889cc-3728-4064-bbcb-5c1c446633b3",
        "type": "regular",
        "name": "Test campaign",
        "subject": "Hello, world",
        "from_email": "listmonk \u003cnoreply@listmonk.yoursite.com\u003e",
        "body": "",
        "altbody": null,
        "send_at": null,
        "status": "draft",
        "content_type": "richtext",
        "tags": ["test"],
        "template_id": 1,
        "messenger": "email"
    }
}
```


#### ```PUT``` /api/campaigns/:`campaign_id`/status

Modifies a campaign status to start, pause, cancel, or schedule a campaign.

##### Parameters 

Name              |  Parameter Type         | Data Type                 |    Required/Optional | Description
------------------|-------------------------|---------------------------|----------------------|-----------------------------
`campaign_id`      | Path Parameter          | Number                    | Required             | The id value of the campaign whose status is to be modified.
`status`            | Request Body            | String                    | Required             | `scheduled`, `running`, `paused`, `cancelled`.                        


###### Note: 
 > * Only "scheduled" campaigns can be saved as "draft".
  * Only "draft" campaigns can be "scheduled".
  * Only "paused" campaigns and "draft" campaigns can be started.
  * Only "running" campaigns can be "cancelled" and "paused".


##### Example Request

```shell
curl -u "username:password" -X PUT 'http://localhost:9000/api/campaigns/1/status' \
--header 'Content-Type: application/json' \
--data-raw '{"status":"scheduled"}'
```

##### Example Response

```json
{
    "data": {
        "id": 1,
        "created_at": "2020-03-14T17:36:41.29451+01:00",
        "updated_at": "2020-04-08T19:35:17.331867+01:00",
        "CampaignID": 0,
        "views": 0,
        "clicks": 0,
        "lists": [
            {
                "id": 1,
                "name": "Default list"
            }
        ],
        "started_at": null,
        "to_send": 0,
        "sent": 0,
        "uuid": "57702beb-6fae-4355-a324-c2fd5b59a549",
        "type": "regular",
        "name": "Test campaign",
        "subject": "Welcome to listmonk",
        "from_email": "No Reply <noreply@yoursite.com>",
        "body": "<h3>Hi {{ .Subscriber.FirstName }}!</h3>\n\t\t\tThis is a test e-mail campaign. Your second name is {{ .Subscriber.LastName }} and you are from {{ .Subscriber.Attribs.city }}.",
        "send_at": "2020-03-15T17:36:41.293233+01:00",
        "status": "scheduled",
        "content_type": "richtext",
        "tags": [
            "test-campaign"
        ],
        "template_id": 1,
        "messenger": "email"
    }
}
```

#### ```DELETE``` /api/campaigns/:`campaign_id`

Deletes a campaign, only scheduled campaigns that have not yet been started can be deleted.  

##### Parameters

Name     |  Parameter Type    | Data Type      | Required/Optional   | Description
---------|--------------------|----------------|---------------------|------------------------------
`campaign_id`| Path Parameter   | Number         | Required            | The id value of the campaign you want to delete.


##### Example Request

```shell
curl -u "username:password" -X DELETE 'http://localhost:9000/api/campaigns/34'
```

##### Example Response

```json
{
    "data": true
}
```
