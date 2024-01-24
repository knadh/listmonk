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
| ------ | ---------------- | ---------------------- |
| `POST` | /webhooks/bounce | Record a bounce event. |


| Name            | Type      | Required   | Description                                                                          |
| ----------------| --------- | -----------| ------------------------------------------------------------------------------------ |
| subscriber_uuid | string    |            | The UUID of the subscriber. Either this or `email` is required.                      |
| email           | string    |            | The e-mail of the subscriber. Either this or `subscriber_uuid` is required.          |
| campaign_uuid   | string    |            | UUID of the campaign for which the bounce happened.                                  |
| source          | string    | Yes        | A string indicating the source, eg: `api`, `my_script` etc.                          |
| type            | string    | Yes        | `hard` or `soft` bounce. Currently, this has no effect on how the bounce is treated. |
| meta            | string    |            | An optional escaped JSON string with arbitrary metadata about the bounce event.      |
 

```shell
curl -u 'username:password' -X POST 'http://localhost:9000/webhooks/bounce' \
	-H "Content-Type: application/json" \
	--data '{"email": "user1@mail.com", "campaign_uuid": "9f86b50d-5711-41c8-ab03-bc91c43d711b", "source": "api", "type": "hard", "meta": "{\"additional\": \"info\"}}'

```

## External webhooks
listmonk supports receiving bounce webhook events from the following SMTP providers.

| Endpoint                                                  | Description                            | More info                                                                                                             |
|:----------------------------------------------------------|:---------------------------------------|:----------------------------------------------------------------------------------------------------------------------|
| `https://listmonk.yoursite.com/webhooks/service/ses`      | Amazon (AWS) SES                       | See below                                                                                                             |
| `https://listmonk.yoursite.com/webhooks/service/sendgrid` | Sendgrid / Twilio Signed event webhook | [More info](https://docs.sendgrid.com/for-developers/tracking-events/getting-started-event-webhook-security-features) |
| `https://listmonk.yoursite.com/webhooks/service/postmark` | Postmark webhook                       | [More info](https://postmarkapp.com/developer/webhooks/webhooks-overview)                                             |

## Amazon Simple Email Service (SES)

If using SES as your SMTP provider, automatic bounce processing is the recommended way to maintain your [sender reputation](https://docs.aws.amazon.com/ses/latest/dg/monitor-sender-reputation.html). The settings below are based on Amazon's [recommendations](https://docs.aws.amazon.com/ses/latest/dg/send-email-concepts-deliverability.html). Please note that your sending domain must be verified in SES before proceeding.

1. In listmonk settings, go to the "Bounces" tab and configure the following:
    - Enable bounce processing: `Enabled`
        - Soft:
            - Bounce count: `2`
            - Action: `None`
        - Hard:
            - Bounce count: `1`
            - Action: `Blocklist`
        - Complaint: 
            - Bounce count: `1`
            - Action: `Blocklist`
    - Enable bounce webhooks: `Enabled`
    - Enable SES: `Enabled`
2. In the AWS console, go to [Simple Notification Service](https://console.aws.amazon.com/sns/) and create a new topic with the following settings:
    - Type: `Standard`
    - Name: `ses-bounces` (or any other name)
3. Create a new subscription to that topic with the following settings:
    - Protocol: `HTTPS`
    - Endpoint: `https://listmonk.yoursite.com/webhooks/service/ses`
    - Enable raw message delivery: `Disabled` (unchecked)
4. SES will then make a request to your listmonk instance to confirm the subscription. After a page refresh, the subscription should have a status of "Confirmed". If not, your endpoint may be incorrect or not publicly accessible.
5. In the AWS console, go to [Simple Email Service](https://console.aws.amazon.com/ses/) and click "Verified identities" in the left sidebar.
6. Click your domain and go to the "Notifications" tab.
7. Next to "Feedback notifications", click "Edit".
8. For both "Bounce feedback" and "Complaint feedback", use the following settings:
    - SNS topic: `ses-bounces` (or whatever you named it)
    - Include original email headers: `Enabled` (checked)
9. Bounce processing should now be working. You can test it with [SES simulator addresses](https://docs.aws.amazon.com/ses/latest/dg/send-an-email-from-console.html#send-email-simulator). Add them as subscribers, send them campaign previews, and ensure that the appropriate action was taken after the configured bounce count was reached.
    - Soft bounce: `ooto@simulator.amazonses.com`
    - Hard bounce: `bounce@simulator.amazonses.com`
    - Complaint: `complaint@simulator.amazonses.com`
10. You can optionally [disable email feedback forwarding](https://docs.aws.amazon.com/ses/latest/dg/monitor-sending-activity-using-notifications-email.html#monitor-sending-activity-using-notifications-email-disabling).

## Exporting bounces

Bounces can be exported via the JSON API:
```shell
curl -u 'username:passsword' 'http://localhost:9000/api/bounces'
```

Or by querying the database directly:
```sql
SELECT bounces.created_at,
    bounces.subscriber_id,
    subscribers.uuid AS subscriber_uuid,
    subscribers.email AS email
FROM bounces
LEFT JOIN subscribers ON (subscribers.id = bounces.subscriber_id)
ORDER BY bounces.created_at DESC LIMIT 1000;
```
