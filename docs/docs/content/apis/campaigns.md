# API / Campaigns

| Method | Endpoint                                                                    | Description                               |
|:-------|:----------------------------------------------------------------------------|:------------------------------------------|
| GET    | [/api/campaigns](#get-apicampaigns)                                         | Retrieve all campaigns.                   |
| GET    | [/api/campaigns/{campaign_id}](#get-apicampaignscampaign_id)                | Retrieve a specific campaign.             |
| GET    | [/api/campaigns/{campaign_id}/preview](#get-apicampaignscampaign_idpreview) | Retrieve preview of a campaign.           |
| GET    | [/api/campaigns/running/stats](#get-apicampaignsrunningstats)               | Retrieve stats of specified campaigns.    |
| POST   | [/api/campaigns](#post-apicampaigns)                                        | Create a new campaign.                    |
| POST   | [/api/campaigns/{campaign_id}/test](#post-apicampaignscampaign_idtest)      | Test campaign with arbitrary subscribers. |
| PUT    | [/api/campaigns/{campaign_id}](#put-apicampaignscampaign_id)                | Update a campaign.                        |
| PUT    | [/api/campaigns/{campaign_id}/status](#put-apicampaignscampaign_idstatus)   | Change status of a campaign.              |
| DELETE | [/api/campaigns/{campaign_id}](#delete-apicampaignscampaign_id)             | Delete a campaign.                        |

______________________________________________________________________

#### GET /api/campaigns

Retrieve all campaigns.

##### Example Request

```shell
 curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns?page=1&per_page=100'
```

##### Parameters

| Name     | Type     | Required | Description                                                          |
|:---------|:---------|:---------|:---------------------------------------------------------------------|
| order    | string   |          | Sorting order: ASC for ascending, DESC for descending.               |
| order_by | string   |          | Result sorting field. Options: name, status, created_at, updated_at. |
| query    | string   |          | SQL query expression to filter campaigns.                            |
| status   | []string |          | Status to filter campaigns. Repeat in the query for multiple values. |
| tags     | []string |          | Tags to filter campaigns. Repeat in the query for multiple values.   |
| page     | number   |          | Page number for paginated results.                                   |
| per_page | number   |          | Results per page. Set as 'all' for all results.                      |

##### Example Response

```json
{
    "data": {
        "results": [
            {
                "id": 1,
                "created_at": "2020-03-14T17:36:41.29451+01:00",
                "updated_at": "2020-03-14T17:36:41.29451+01:00",
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

______________________________________________________________________

#### GET /api/campaigns/{campaign_id}

Retrieve a specific campaign.

##### Parameters

| Name        | Type      | Required | Description  |
|:------------|:----------|:---------|:-------------|
| campaign_id | number    | Yes      | Campaign ID. |

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns/1'
```

##### Example Response

```json
{
    "data": {
        "id": 1,
        "created_at": "2020-03-14T17:36:41.29451+01:00",
        "updated_at": "2020-03-14T17:36:41.29451+01:00",
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

______________________________________________________________________

#### GET /api/campaigns/{campaign_id}/preview

Preview a specific campaign.

##### Parameters

| Name        | Type      | Required | Description             |
|:------------|:----------|:---------|:------------------------|
| campaign_id | number    | Yes      | Campaign ID to preview. |

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns/1/preview'
```

##### Example Response

```html
<h3>Hi John!</h3>
This is a test e-mail campaign. Your second name is Doe and you are from Bengaluru.
```

______________________________________________________________________

#### GET /api/campaigns/running/stats

Retrieve stats of specified campaigns.

##### Parameters

| Name        | Type      | Required | Description                    |
|:------------|:----------|:---------|:-------------------------------|
| campaign_id | number    | Yes      | Campaign IDs to get stats for. |

##### Example Request

```shell
curl -u "username:password" -X GET 'http://localhost:9000/api/campaigns/running/stats?campaign_id=1'
```

##### Example Response

```json
{
    "data": []
}
```

______________________________________________________________________

#### POST /api/campaigns

Create a new campaign.

##### Parameters

| Name         | Type      | Required | Description                                                                             |
|:-------------|:----------|:---------|:----------------------------------------------------------------------------------------|
| name         | string    | Yes      | Campaign name.                                                                          |
| subject      | string    | Yes      | Campaign email subject.                                                                 |
| lists        | number\[\]  | Yes      | List IDs to send campaign to.                                                           |
| from_email   | string    |          | 'From' email in campaign emails. Defaults to value from settings if not provided.       |
| type         | string    | Yes      | Campaign type: 'regular' or 'optin'.                                                    |
| content_type | string    | Yes      | Content type: 'richtext', 'html', 'markdown', 'plain'.                                  |
| body         | string    | Yes      | Content body of campaign.                                                               |
| altbody      | string    |          | Alternate plain text body for HTML (and richtext) emails.                               |
| send_at      | string    |          | Timestamp to schedule campaign. Format: 'YYYY-MM-DDTHH:MM:SS'.                          |
| messenger    | string    |          | 'email' or a custom messenger defined in settings. Defaults to 'email' if not provided. |
| template_id  | number    |          | Template ID to use. Defaults to default template if not provided.                       |
| tags         | string\[\]  |          | Tags to mark campaign.                                                                  |
| headers      | JSON      |          | Key-value pairs to send as SMTP headers. Example: \[{"x-custom-header": "value"}\].       |

##### Example request

```shell
curl -u "username:password" 'http://localhost:9000/api/campaigns' -X POST -H 'Content-Type: application/json;charset=utf-8' --data-raw '{"name":"Test campaign","subject":"Hello, world","lists":[1],"from_email":"listmonk <noreply@listmonk.yoursite.com>","content_type":"richtext","messenger":"email","type":"regular","tags":["test"],"template_id":1}'
```

##### Example response

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

______________________________________________________________________

#### POST /api/campaigns/{campaign_id}/test

Test campaign with arbitrary subscribers.

Use the same parameters in [POST /api/campaigns](#post-apicampaigns) in addition to the below parameters.

##### Parameters

| Name        | Type     | Required | Description                                        |
|:------------|:---------|:---------|:---------------------------------------------------|
| subscribers | string\[\] | Yes      | List of subscriber e-mails to send the message to. |

______________________________________________________________________

#### PUT /api/campaigns/{campaign_id}

Update a campaign.

> Refer to parameters from [POST /api/campaigns](#post-apicampaigns)

______________________________________________________________________

#### PUT /api/campaigns/{campaign_id}

Update a specific campaign.

> Refer to parameters from [POST /api/campaigns](#post-apicampaigns)

______________________________________________________________________

#### PUT /api/campaigns/{campaign_id}/status

Change status of a campaign.

##### Parameters

| Name        | Type      | Required | Description                                                             |
|:------------|:----------|:---------|:------------------------------------------------------------------------|
| campaign_id | number    | Yes      | Campaign ID to change status.                                           |
| status      | string    | Yes      | New status for campaign: 'scheduled', 'running', 'paused', 'cancelled'. |

##### Note

> - Only 'scheduled' campaigns can change status to 'draft'.
> - Only 'draft' campaigns can change status to 'scheduled'.
> - Only 'paused' and 'draft' campaigns can start ('running' status).
> - Only 'running' campaigns can change status to 'cancelled' and 'paused'.

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

______________________________________________________________________

#### DELETE /api/campaigns/{campaign_id}

Delete a campaign.

##### Parameters

| Name        | Type      | Required | Description            |
|:------------|:----------|:---------|:-----------------------|
| campaign_id | number    | Yes      | Campaign ID to delete. |

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
