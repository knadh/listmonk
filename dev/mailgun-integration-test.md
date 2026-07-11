# Mailgun Bounce Webhook Integration Test (End-to-End)

This procedure verifies end-to-end Mailgun bounce processing against a real Mailgun account and a running listmonk instance.

## Prerequisites

- Mailgun account with a sandbox domain.
- At least one authorized recipient on that sandbox domain.
- Local listmonk checkout with Mailgun bounce support.
- Public HTTPS tunnel to local listmonk (`ngrok`, `cloudflared`, or equivalent).

## Mailgun Setup

1. In Mailgun, open your domain settings and copy the HTTP webhook signing key from API security settings.
2. Configure webhooks for bounce-related events and point each to:
   - `https://<public-tunnel-host>/webhooks/service/mailgun`
3. Subscribe these events at minimum:
   - `permanent_failure`
   - `temporary_failure`
   - `complained`

Use the webhook signing key, not the Mailgun sending/API key.

## listmonk Setup

1. Run DB upgrades so `bounce.mailgun` exists:
   - `./listmonk --upgrade`
2. Start listmonk locally (default: `http://localhost:9000`).
3. In admin settings under bounces, enable Mailgun and set `bounce.mailgun.key` to the Mailgun webhook signing key.
4. Start your HTTPS tunnel for local port 9000 (example: `ngrok http 9000`) and update Mailgun webhook URLs with the active tunnel URL.

## Trigger Events

1. Use Mailgun’s webhook UI "send test webhook" action for:
   - `permanent_failure`
   - `temporary_failure`
   - `complained`
2. Real hard bounce test:
   - Send to `bounce@simulator.mailgun.org`.
3. Real complaint test:
   - Use Mailgun complaint simulator if available on your plan, or mark a received test email as spam in a real inbox.

For campaign linkage checks, include campaign UUID metadata when sending:
- API send: `v:X-Listmonk-Campaign=<campaign-uuid>`
- SMTP send: `X-Mailgun-Variables: {"X-Listmonk-Campaign":"<campaign-uuid>"}`

## Expected Outcomes

- listmonk accepts webhook requests with HTTP 200.
- A bounce row is created with:
  - `source = mailgun`
  - correct bounce `type` (`hard`, `soft`, `complaint`)
  - linked campaign when `X-Listmonk-Campaign` is present
- Negative check:
  1. Set an incorrect `bounce.mailgun.key` in listmonk.
  2. Re-send a Mailgun test webhook.
  3. Request fails with HTTP 400 and no bounce is recorded.

## Teardown

- Disable Mailgun bounce webhook settings in listmonk or clear the key.
- Remove webhook URLs from Mailgun (or disable them) to avoid stray events hitting your dev tunnel.

## Troubleshooting

- Signature failures: verify you are using the webhook signing key, not the API key.
- No webhook hits: verify tunnel is active and Mailgun URLs use HTTPS.
- Missing campaign linkage: verify `X-Listmonk-Campaign` is present in `user-variables` or message headers.
