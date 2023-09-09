# API / Transactional

| Method | Endpoint | Description |
|:-------|:---------|:------------|
| `POST` | /api/tx  |             |


## POST /api/tx
Send a transactional message to one or multiple subscribers using a predefined transactional template.


##### Parameters
| Name                | Data Type | Optional | Description                                                                                                                                      |
|:--------------------|:----------|:--------|:-------------------------------------------------------------------------------------------------------------------------------------------------|
| `subscriber_email`  | String    | Optional | E-mail of the subscriber. Either this or `subscriber_id` should be passed.                                                                       |
| `subscriber_id`     | Number    | Optional | ID of the subscriber. Either this or `subscriber_email` should be passed.                                                                        |
| `subscriber_emails` | []String  | Optional | E-mails of the subscribers. This is an alternative to `subscriber_email` for multiple recipients. `["email1@example.com", "emailX@example.com"]` |
| `subscriber_ids`    | []Number  | Optional | IDs of the subscribers. This is an alternative to `subscriber_id` for multiple recipients. `[1,2,3]`                                              |
| `template_id`       | Number    | Required | ID of the transactional template to use in the message.                                                                                          |
| `from_email`        | String    | Optional | Optional `from` email. eg: `Company <email@company.com>`                                                                                         |
| `data`              | Map       | Optional | Optional data in `{}` nested map. Available in the template as `{{ .Tx.Data.* }}`                                                                |
| `headers`           | []Map     | Optional | Optional array of mail headers. `[{"key": "value"}, {"key": "value"}]`                                                                           |
| `messenger`         | String    | Optional | Messenger to use to send the message. Default value is `email`.                                                                                  |
| `content_type`      | String    | Optional | `html`, `markdown`, `plain`                                                                                                                      |


##### Request
```shell
curl -u "username:password" "http://localhost:9000/api/tx" -X POST \
     -H 'Content-Type: application/json; charset=utf-8' \
     --data-binary @- << EOF
    {
        "subscriber_email": "user@test.com",
        "template_id": 2,
        "data": {"order_id": "1234", "date": "2022-07-30", "items": [1, 2, 3]},
        "content_type": "html"
    }
EOF
```

##### Response
``` json
{
    "data": true
}
```

### File Attachments
To include file attachments in a transactional message, use Content-Type `multipart/form-data`.
Use the parameters described above as a JSON object via the `data` form key and include an arbitrary number of attachments via the `file` key.

```shell
curl -u "username:password" "http://localhost:9000/api/tx" -X POST \
-F 'data=\"{
    \"subscriber_email\": \"user@test.com\",
    \"template_id\": 4
}"' \
-F 'file=@"/path/to/attachment.pdf"' \
-F 'file=@"/path/to/attachment2.pdf"'
```

