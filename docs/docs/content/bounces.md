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

### Bounce classification
listmonk applies a series of heuristics looking for keywords in the bounced mail body to guess if it is a 'soft' bounce or a 'hard' bounce. For instance, 4.x.x and 5.x.x error status codes, common strings such as "mailbox not found" etc. If none of the heuristics match, then the bounce mail is considered to be 'soft' by default.

## Webhook API
The bounce webhook API can be used to record bounce events with custom scripting. This could be by reading a mailbox, a database, or mail server logs.

| Method | Endpoint         | Description            |
| ------ | ---------------- | ---------------------- |
| `POST` | /webhooks/bounce | Record a bounce event. |


| Name            | Type   | Required | Description                                                                          |
| --------------- | ------ | -------- | ------------------------------------------------------------------------------------ |
| subscriber_uuid | string |          | The UUID of the subscriber. Either this or `email` is required.                      |
| email           | string |          | The e-mail of the subscriber. Either this or `subscriber_uuid` is required.          |
| campaign_uuid   | string |          | UUID of the campaign for which the bounce happened.                                  |
| source          | string | Yes      | A string indicating the source, eg: `api`, `my_script` etc.                          |
| type            | string | Yes      | `hard` or `soft` bounce. Currently, this has no effect on how the bounce is treated. |
| meta            | string |          | An optional escaped JSON string with arbitrary metadata about the bounce event.      |
 

```shell
curl -u 'api_username:access_token' -X POST 'http://localhost:9000/webhooks/bounce' \
	-H "Content-Type: application/json" \
	--data '{"email": "user1@mail.com", "campaign_uuid": "9f86b50d-5711-41c8-ab03-bc91c43d711b", "source": "api", "type": "hard", "meta": "{\"additional\": \"info\"}}'

```

## External webhooks
listmonk supports receiving bounce webhook events from the following SMTP providers.

| Endpoint                                                      | Description                            | More info                                                                                                             |
|:--------------------------------------------------------------|:---------------------------------------|:----------------------------------------------------------------------------------------------------------------------|
| `https://listmonk.yoursite.com/webhooks/service/ses`          | Amazon (AWS) SES                       | See below                                                                                                             |
| `https://listmonk.yoursite.com/webhooks/service/azure`        | Azure Communication Services (ACS)     | [More info](https://learn.microsoft.com/en-us/azure/event-grid/communication-services-email-events)                 |
| `https://listmonk.yoursite.com/webhooks/service/sendgrid`     | Sendgrid / Twilio Signed event webhook | [More info](https://docs.sendgrid.com/for-developers/tracking-events/getting-started-event-webhook-security-features) |
| `https://listmonk.yoursite.com/webhooks/service/postmark`     | Postmark webhook                       | [More info](https://postmarkapp.com/developer/webhooks/webhooks-overview)                                             |
| `https://listmonk.yoursite.com/webhooks/service/forwardemail` | Forward Email webhook                  | [More info](https://forwardemail.net/en/faq#do-you-support-bounce-webhooks)                                           |
| `https://listmonk.yoursite.com/webhooks/service/lettermint`   | Lettermint webhook                     | [More info](https://lettermint.co/knowledge-base/guides/send-newsletter-with-listmonk)                                                |
| `https://listmonk.yoursite.com/webhooks/service/anypost`      | Anypost webhook                        | [More info](https://anypost.com/docs/webhooks)                                                                        |

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
5. In the AWS console, go to [Simple Email Service](https://console.aws.amazon.com/ses/) and click "Identities" in the left sidebar.
6. Click your domain and go to the "Notifications" tab.
7. Next to "Feedback notifications", click "Edit".
8. For both "Bounce feedback" and "Complaint feedback", use the following settings:
    - SNS topic: `ses-bounces` (or whatever you named it)
    - Include original email headers: `Enabled` (checked)
9. Repeat steps 6-8 for any `Email address` identities you send from using listmonk
10. Bounce processing should now be working. You can test it with [SES simulator addresses](https://docs.aws.amazon.com/ses/latest/dg/send-an-email-from-console.html#send-email-simulator). Add them as subscribers, send them campaign previews, and ensure that the appropriate action was taken after the configured bounce count was reached.
    - Soft bounce: `ooto@simulator.amazonses.com`
    - Hard bounce: `bounce@simulator.amazonses.com`
    - Complaint: `complaint@simulator.amazonses.com`
11. You can optionally [disable email feedback forwarding](https://docs.aws.amazon.com/ses/latest/dg/monitor-sending-activity-using-notifications-email.html#monitor-sending-activity-using-notifications-email-disabling).

## Azure Communication Services (ACS)

If you use Azure Communication Services Email, listmonk can receive delivery report events from Azure Event Grid and turn them into bounces.

1. In listmonk settings, go to "Bounces" and configure:
    - Enable bounce processing: `Enabled`
    - Enable bounce webhooks: `Enabled`
    - Enable Azure ACS: `Enabled`
    - Optional: set `Azure Event Grid Shared Secret`.
    - Optional: set `Azure Shared Secret Header Name` if you want listmonk to read the secret from a header (defaults to `X-Listmonk-Webhook-Secret`).
2. In listmonk settings, go to "SMTP" and use the `Azure ACS` quick preset to fill SMTP defaults.
3. In Azure, create an Event Grid subscription for your ACS Email events with:
    - Endpoint type: `Web Hook`
    - Endpoint URL: `https://listmonk.yoursite.com/webhooks/service/azure`
      - If using query-param auth, append `?code=<your-shared-secret>`.
      - If using header auth, configure Event Grid to include the same secret in the header name configured in listmonk.
4. During subscription creation, Event Grid sends a subscription validation event. listmonk automatically returns `validationResponse` for this handshake.
5. Subscribe to `Microsoft.Communication.EmailDeliveryReportReceived` events. listmonk maps relevant statuses to bounce records.
6. Send test mail and verify bounces in listmonk.

## Anypost

If you use [Anypost](https://anypost.com) as your SMTP provider, listmonk can receive its webhook events and turn them into bounces. listmonk maps `email.bounced` to hard bounces (soft when the failure is transient) and `email.complained` to complaints. It also maps `email.suppressed` (a send Anypost dropped because the address was already on your suppression list): those carrying `data.suppression.reason` of `permanent_bounce` become hard bounces and `complaint` become complaints, keeping listmonk in sync even if it never saw the original event. Suppressions for `unsubscribed` and `manual` reasons are opt-outs, not bounces, and are ignored. Bounces are attributed to campaigns automatically: Anypost echoes the `X-Listmonk-Campaign` header on its events. Complaints arriving via feedback loops and out-of-band bounces carry no headers, so those are recorded against the subscriber without a campaign.

1. In the [Anypost dashboard](https://anypost.com), create a webhook endpoint with:
    - URL: `https://listmonk.yoursite.com/webhooks/service/anypost`
    - Events: `email.bounced`, `email.complained`, `email.suppressed`
2. Copy the webhook's signing secret (`whsec_...`), shown once at creation.
3. Enable the integration via the listmonk [settings API](https://listmonk.app/docs/apis/apis/) (the admin UI does not have fields for it yet). The API takes the full settings object, so fetch it, blank out the masked secrets (listmonk preserves secrets submitted as empty strings), set the Anypost keys, and PUT it back:

```shell
curl -su 'api_username:access_token' 'http://localhost:9000/api/settings' \
	| jq '.data
		| walk(if type == "string" and test("^•+$") then "" else . end)
		| .["bounce.enabled"] = true
		| .["bounce.webhooks_enabled"] = true
		| .["bounce.anypost"] = {"enabled": true, "key": "whsec_..."}' \
	| curl -u 'api_username:access_token' -X PUT 'http://localhost:9000/api/settings' \
		-H 'Content-Type: application/json' --data @-
```

4. Send test mail and verify bounces in listmonk.

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
