# Configuring Resend with listmonk

Resend works with listmonk via its SMTP interface. Since listmonk is provider-agnostic, you can configure Resend just like any other SMTP service.

## Prerequisites
1. A [Resend](https://resend.com) account.
2. A verified domain in your Resend dashboard.
3. A Resend API Key with "Sending" permissions.

## SMTP Configuration Steps

1. Log in to your listmonk dashboard.
2. Navigate to **Settings -> SMTP**.
3. Click **Add new server** and enter the following details:

| Field | Value |
| :--- | :--- |
| **Name** | `Resend` |
| **Host** | `smtp.resend.com` |
| **Port** | `465` (Recommended) or `587` |
| **Auth Protocol** | `login` |
| **Username** | `resend` |
| **Password** | Your Resend API Key (`re_...`) |
| **TLS Type** | `TLS` (for port 465) or `STARTTLS` (for port 587) |
| **Max Conns** | `10`–`25` (Resend handles high concurrency well) |

4. Click **Test connection** to verify listmonk can communicate with Resend.
5. Use the **Send test email** button to ensure your "From" address is correctly verified.

## Important Note on Bounces
Resend works perfectly for **sending** via SMTP. However, listmonk does not currently have a native "one-click" webhook handler for Resend's bounce events (unlike Amazon SES or SendGrid).

- **Sending:** Fully supported via SMTP.
- **Bounce Tracking:** You can monitor bounces in the Resend dashboard. 
- **Automatic Sync:** To sync bounces back to listmonk automatically, you would need a small middleware to transform Resend's webhook payload into listmonk's generic bounce format and POST it to the `/webhooks/service/` endpoint.
