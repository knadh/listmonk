# API / Transactional

| Method | Endpoint | Description                 |
| :----- | :------- | :-------------------------- |
| POST   | /api/tx  | Send transactional messages |

______________________________________________________________________

#### POST /api/tx

Allows sending transactional messages to one or more subscribers via a preconfigured transactional template.

##### Parameters

| Name              | Type       | Required | Description                                                                |
| :---------------- | :--------- | :------- | :------------------------------------------------------------------------- |
| subscriber_email  | string     |          | Email of the subscriber. Can substitute with `subscriber_id`.              |
| subscriber_id     | number     |          | Subscriber's ID can substitute with `subscriber_email`.                    |
| subscriber_emails | string\[\] |          | Multiple subscriber emails as alternative to `subscriber_email`.           |
| subscriber_ids    | number\[\] |          | Multiple subscriber IDs as an alternative to `subscriber_id`.              |
| subscriber_mode   | string     |          | Subscriber lookup mode: `default`, `fallback`, or `external`               |
| template_id       | number     | Yes      | ID of the transactional template to be used for the message.               |
| from_email        | string     |          | Optional sender email.                                                     |
| subject           | string     |          | Optional subject. If empty, the subject defined on the template is used    |
| data              | JSON       |          | Optional nested JSON map. Available in the template as `{{ .Tx.Data.* }}`. |
| headers           | JSON\[\]   |          | Optional array of email headers.                                           |
| messenger         | string     |          | Messenger to send the message. Default is `email`.                         |
| content_type      | string     |          | Email format options include `html`, `markdown`, and `plain`.              |

##### Subscriber modes

The `subscriber_mode` parameter controls how the recipients (subscribers or non-subscriber recipients) are resolved.

| Mode       | Description                                                                                                                                                                                                                                                                    |
| :--------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `default`  | Recipients must exist as subscribers in the database. Pass either `subscriber_emails` or `subscriber_ids`.                                                                                                                                                                     |
| `fallback` | Only accepts `subscriber_emails` and looks up subscribers in the database. If not found, sends the message to the e-mail anyway. In the template, apart from `{{ .Subscriber.Email }}`, other subscriber fields such as `.Name`. will be empty. Use `{{ Tx.Data.* }}` instead. |
| `external` | Sends to the given `subscriber_emails` without subscriber lookup in the database. In the template, apart from `{{ .Subscriber.Email }}`, other subscriber fields such as `.Name`. will be empty. Use `{{ Tx.Data.* }}` instead.                                                |

##### Example

```shell
curl -u "api_user:token" "http://localhost:9000/api/tx" -X POST \
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

##### Example with external mode

Send to arbitrary email addresses without requiring them to be subscribers:

```shell
curl -u "api_user:token" "http://localhost:9000/api/tx" -X POST \
     -H 'Content-Type: application/json; charset=utf-8' \
     --data-binary @- << EOF
    {
        "subscriber_mode": "external",
        "subscriber_emails": ["recipient@example.com"],
        "template_id": 2,
        "data": {"name": "John", "order_id": "1234"},
        "content_type": "html"
    }
EOF
```

In the template, use `{{ .Tx.Data.name }}`, `{{ .Tx.Data.order_id }}`, etc. to access the data.

______________________________________________________________________

#### File Attachments

To include file attachments in a transactional message, use the `multipart/form-data` Content-Type. Use `data` param for the parameters described above as a JSON object. Include any number of attachments via the `file` param.

```shell
curl -u "api_user:token" "http://localhost:9000/api/tx" -X POST \
-F 'data=\"{
    \"subscriber_email\": \"user@test.com\",
    \"template_id\": 4
}"' \
-F 'file=@"/path/to/attachment.pdf"' \
-F 'file=@"/path/to/attachment2.pdf"'
```
