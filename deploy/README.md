# Deploying this fork

This is a fork of listmonk with extra backend features (e.g. per-campaign subscriber
segments). It is deployed to the prealpha `selfhosted-tools` stack (Docker Compose + Traefik
on a VM). There is **no registry**: a self-contained image is built locally and shipped to the
server over SSH.

## One-time: point the stack at the fork image

In `selfhosted-tools/services/listmonk.yml`:

```yaml
  listmonk:
    image: listmonk-fork:prod      # was: listmonk/listmonk:latest
    pull_policy: never             # use the locally-loaded image, never a registry
    # ... unchanged ...
    labels:
      # remove: com.centurylinklabs.watchtower.scope=auto-update
```

Commit and push to `main` so it goes through the normal GitLab pipeline. `pull_policy: never`
means a deploy fails loudly if the image was not shipped first (instead of pulling upstream).

## Each deploy

1. **Back up the DB** (`selfhosted-tools/scripts/backup-restore.sh`).
2. **Build + ship** the image from this checkout:
   ```sh
   DEPLOY_HOST=<server-host> ./deploy/ship-fork-image.sh
   ```
   Needs Docker (buildx), Go, and Node + Yarn on the build machine. It cross-compiles a
   linux/amd64 self-contained binary, builds `listmonk-fork:prod`, and `docker save | ssh
   docker load`s it onto the server. Add `RUN_DEPLOY=1` to also run the remote `deploy.sh`.
3. **Roll out** (if not using `RUN_DEPLOY=1`): trigger the normal selfhosted-tools deploy
   (push to `main`), or on the server `docker compose ... up -d listmonk`. The container's
   command runs `--upgrade`, applying pending migrations automatically.

## Verify

- Admin UI footer shows `v6.3.0-fork+<sha>`.
- New campaign form shows the **Segment** field and **Preview recipients** button.

## Notes

- The `subscriber_query` migration is `v6.3.0` and is an `ADD COLUMN IF NOT EXISTS` (nullable):
  instant and non-destructive even on large subscriber tables.
- Fork maintenance: if a future upstream release also ships a `v6.3.0` migration, reconcile it
  (ours is idempotent and safe to re-run; rename if you need upstream's `v6.3.0` to still apply).
- `ship-fork-image.sh` overwrites `listmonk-fork:prod` each run; the old image becomes dangling
  and is cleaned by the server's `docker image prune -f`. The `:<sha>` tags accumulate; prune
  occasionally.
