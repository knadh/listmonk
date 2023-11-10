# API / Transactional

| Method | Endpoint | Description                    |
|:-------|:---------|:-------------------------------|
| POST   | /api/tx  | Send transactional messages    |

______________________________________________________________________

#### POST /api/tx

Allows sending transactional messages to one or more subscribers via a preconfigured transactional template.

##### Parameters

| Name              | Type      | Required | Description                                                                |
|:------------------|:----------|:---------|:---------------------------------------------------------------------------|
| subscriber_email  | string    |          | Email of the subscriber. Can substitute with `subscriber_id`.              |
| subscriber_id     | number    |          | Subscriber's ID can substitute with `subscriber_email`.                    |
| subscriber_emails | string\[\]  |          | Multiple subscriber emails as alternative to `subscriber_email`.           |
| subscriber_ids    | number\[\]  |          | Multiple subscriber IDs as an alternative to `subscriber_id`.              |
| template_id       | number    | Yes      | ID of the transactional template to be used for the message.               |
| from_email        | string    |          | Optional sender email.                                                     |
| data              | JSON      |          | Optional nested JSON map. Available in the template as `{{ .Tx.Data.* }}`. |
| headers           | JSON\[\]    |          | Optional array of email headers.                                           |
| messenger         | string    |          | Messenger to send the message. Default is `email`.                         |
| content_type      | string    |          | Email format options include `html`, `markdown`, and `plain`.              |

##### Example

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

##### Example response

```json
{
    "data": true
}
```

______________________________________________________________________

#### File Attachments

To include file attachments in a transactional message, use the `multipart/form-data` Content-Type. Use `data` param for the parameters described above as a JSON object. Include any number of attachments via the `file` param.

```shell
curl -u "username:password" "http://localhost:9000/api/tx" -X POST \
-F 'data=\"{
    \"subscriber_email\": \"user@test.com\",
    \"template_id\": 4
}"' \
-F 'file=@"/path/to/attachment.pdf"' \
-F 'file=@"/path/to/attachment2.pdf"'
```
