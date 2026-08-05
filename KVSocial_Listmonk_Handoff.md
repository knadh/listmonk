# KVSocial Listmonk Handoff

## Overview

KVSocial is running a customized fork of Listmonk at:

- Production URL: `https://email.kvsocial.com`
- Admin URL: `https://email.kvsocial.com/admin/`
- Local customized project: `/Users/shrestsav/kyvio/projects/listmonk-kvs`

The customized fork is kept separately from the original Listmonk checkout so that future features and fixes can be developed internally without changing the upstream version.

## Production operations documentation

Day-to-day production procedures for Listmonk are maintained in the [KVSocial team knowledge base](https://github.com/KVSocial/team-knowledge/tree/main/Listmonk).

This repository documents the fork, release, deployment, upgrade, and rollback workflow. Use the team knowledge base for production backups, monitoring, incident response, access, and server operations.

## Local development setup

The local development environment is defined in `dev/docker-compose.yml`. It uses a bind mount of the repository, so source changes are available inside the containers without rebuilding the entire application.

The development stack contains:

- Go backend on `http://localhost:9000`
- Frontend development server on `http://localhost:8080`
- PostgreSQL on `localhost:5432`
- MailHog SMTP server on `localhost:1025`
- MailHog web UI on `http://localhost:8025`
- Adminer on `http://localhost:8070`

The backend uses `dev/config.toml` and connects to the Compose PostgreSQL service using the local development database `listmonk-dev`. The frontend connects to the backend through the Docker service name `backend`.

From the customized project directory, use:

```bash
make init-dev-docker   # Build the images and initialize the development database.
make dev-docker        # Start the backend, frontend, PostgreSQL, MailHog, and Adminer.
```

To stop the development stack and remove its database volume:

```bash
make rm-dev-docker
```

At the time this handoff was updated, Docker Desktop/the Docker daemon was not running, so the local containers were not active. Production remains available separately at `https://email.kvsocial.com`.

## Repository and deployment workflow

- `origin` is the KVSocial fork: `KVSocial/listmonk`.
- `upstream` is the official `knadh/listmonk` repository and is fetch-only.
- `kvs-main` is the reviewed KVSocial production branch.
- New work should use a feature branch from `kvs-main`, then go through a pull request and CI.
- Production should use an immutable KVSocial release tag, such as `v6.2.0-kvs.1`, published to GHCR. Do not deploy a moving `kvs-latest` tag or patch a running container directly.
- Take a PostgreSQL backup before release, migration, or upstream-version changes.
- After deployment, check the Listmonk health endpoint, admin login, lists, subscribers, campaigns, templates, and logs.

The detailed release, rollback, upstream-merge, backup, and security procedures are in [`docs/KVS_RELEASES.md`](docs/KVS_RELEASES.md).

## Important environment distinction

The MCP process runs locally as a local stdio process, but the configured MCP connection calls the production Listmonk API at `https://email.kvsocial.com`. It is not automatically connected to the local Docker development stack. Keep the MCP API key outside Git, this document, Asana, and static configuration files, and keep tool approval enabled.

The production database, SendGrid credentials, webhook credentials, GHCR credentials, and admin password must remain in approved secret storage. Never commit them to this repository.

## Completed setup

- Deployed the customized Listmonk version to production.
- Connected Listmonk to the KVSocial SendGrid account.
- Configured branded sending and link-tracking domains.
- Configured and validated SPF, DKIM, and DMARC.
- Confirmed SPF, DKIM, and DMARC passed in Gmail's original-message headers.
- Confirmed TLS-encrypted delivery.
- Configured Listmonk campaign tracking and unsubscribe handling.
- Created the `Claudia` API user with the `MCP Production` role.
- Configured the Listmonk MCP connection for local use.

## Tests completed

- Delivered test campaigns successfully to Gmail and KVSocial/Zoho inboxes.
- Confirmed Gmail delivery to the Promotions tab, which is normal for marketing email.
- Tested click tracking successfully.
- Tested open tracking. Open counts can vary because Gmail uses image proxying and caching, and aliases may belong to the same mailbox.
- Tested the View in Browser link.
- Tested one-click unsubscribe and confirmed the subscriber status changes to `Unsubscribed`.
- Tested bounce processing and confirmed bounced recipients are recorded and blocklisted.
- Tested reply routing from Gmail to Zoho Desk; a support ticket was created successfully (`#83708`).
- Validated MCP authentication, health checks, list retrieval, and production read operations.
- Created a campaign draft through MCP.
- Created, started, and completed Campaign ID 11 entirely through MCP.

## MCP usage

The MCP adapter is the community-maintained [`rhnvrm/listmonk-mcp`](https://github.com/rhnvrm/listmonk-mcp) project. It can be used with Claude Desktop or Codex to inspect lists and subscribers and to create and send campaigns.

Use the `Claudia` API username with the API key as the MCP password. Do not put the API key in this repository or in documentation. The key is stored separately in the approved secure channel/notes.

The Listmonk SDK/API documentation is available at:

https://listmonk.app/docs/apis/sdks/

## Remaining work

- Test the remaining SMTP integration/path.
- Restrict the `Claudia` API user to only the required production mailing lists before wider campaign use.
- Confirm the SendGrid plan supports the expected volume, including approximately 150,000 messages per month for 5,000 messages per day.
- Confirm whether SendGrid is using a shared or dedicated IP.
- Configure Google Postmaster Tools before scaling campaign volume.
- Gradually warm up sender reputation before moving from test traffic to approximately 5,000 messages per day.
- Continue monitoring bounces, complaints, deferrals, domain/IP reputation, and unsubscribe processing.

## Important operational notes

- SPF, DKIM, and DMARC passing proves authentication, but it does not guarantee inbox placement at higher volumes.
- Gmail Promotions placement is still successful inbox delivery; Primary-tab placement is not guaranteed for promotional campaigns.
- Do not send to purchased or scraped lists. Use opted-in subscribers and suppress unsubscribes, spam complaints, hard bounces, and chronically inactive recipients.
- Keep marketing and transactional traffic separated when volume increases, preferably through separate SendGrid streams/IP pools.
- Review the related Asana parent task and subtasks for detailed test comments and evidence.
