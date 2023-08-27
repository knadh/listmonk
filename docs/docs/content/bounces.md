# Bounce processing

Enable bounce processing in Settings -> Bounces. POP3 bounce scanning and APIs only become available once the setting is enabled.

## POP3 bounce mailbox
Configure the bounce mailbox in Settings -> Bounces. Either the "From" e-mail that is set on a campaign (or in settings) should have a POP3 mailbox behind it to receive bounce e-mails, or you should configure a dedicated POP3 mailbox and add that address as the `Return-Path` (envelope sender) header in Settings -> SMTP -> Custom headers box. For example:

```
[
	{"Return-Path": "your-bounce-inbox@site.com"}
]

```

Some mail servers may also return the bounce to the `Reply-To` address, which can also be added to the header settings.

## Webhook API
The bounce webhook API can be used to record bounce events with custom scripting. This could be by reading a mailbox, a database, or mail server logs.

| Method | Endpoint         | Description            |
|--------|------------------|------------------------|
| `POST` | /webhooks/bounce | Record a bounce event. |


| Name              | Data type | Required/Optional | Description                                                                          |
|-------------------|-----------|-------------------|--------------------------------------------------------------------------------------|
| `subscriber_uuid` | String    | Optional          | The UUID of the subscriber. Either this or `email` is required.                      |
| `email`           | String    | Optional          | The e-mail of the subscriber. Either this or `subscriber_uuid` is required.          |
| `campaign_uuid`   | String    | Optional          | UUID of the campaign for which the bounce happened.                                  |
| `source`          | String    | Required          | A string indicating the source, eg: `api`, `my_script` etc.                          |
| `type`            | String    | Required          | `hard` or `soft` bounce. Currently, this has no effect on how the bounce is treated. |
| `meta`            | String    | Optional          | An optional escaped JSON string with arbitrary metadata about the bounce event.      |
 

```shell
curl -u 'username:password' -X POST localhost:9000/webhooks/bounce \
	-H "Content-Type: application/json" \
	--data '{"email": "user1@mail.com", "campaign_uuid": "9f86b50d-5711-41c8-ab03-bc91c43d711b", "source": "api", "type": "hard", "meta": "{\"additional\": \"info\"}}'

```

## External webhooks
listmonk supports receiving bounce webhook events from the following SMTP providers.

| Endpoint                    | Description      | More info |
|-----------------------------|------------------|-----------|
| `https://listmonk.yoursite.com/webhooks/service/ses`      | Amazon (AWS) SES | You can use these [Mautic steps](https://docs.mautic.org/en/channels/emails/bounce-management#amazon-webhook) as a general guide, but use your listmonk's endpoint instead. <ul>  <li>When creating the *topic* select "standard" instead of the preselected "FIFO". You can put a name and leave everything else at default.</li>  <li>When creating a *subscription* choose HTTPS for "Protocol", and leave *"Enable raw message delivery"* UNCHECKED.</li>  <li>On the _"SES -> verified identities"_ page, make sure to check **"[include original headers](https://github.com/knadh/listmonk/issues/720#issuecomment-1046877192)"**.</li>  <li>The Mautic screenshot suggests you should turn off _email feedback forwarding_, but that's completely optional depending on whether you want want email notifications.</li></ul>   |
| `https://listmonk.yoursite.com/webhooks/service/sendgrid` | Sendgrid / Twilio Signed event webhook         | [More info](https://docs.sendgrid.com/for-developers/tracking-events/getting-started-event-webhook-security-features) |



## Verification

If you're using Amazon SES you can use Amazon's test emails to make sure everything's working: [https://docs.aws.amazon.com/ses/latest/dg/send-an-email-from-console.html](https://docs.aws.amazon.com/ses/latest/dg/send-an-email-from-console.html)
```
success@simulator.amazonses.com
bounce@simulator.amazonses.com
complaint@simulator.amazonses.com
suppressionlist@simulator.amazonses.com
```
They all count as _hard_ bounces. 


**Exporting bounces**: [https://github.com/knadh/listmonk/issues/863](https://github.com/knadh/listmonk/issues/863)


