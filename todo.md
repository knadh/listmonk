# TODO

## Listmonk (Current Project)

### 🚀 Deployment & Infra
- [ ] **Execute Remote Build**: Run `./deploy_dev.sh` to sync local changes and build the `listmonk-custom` image on `dev.sulopuis.to`.
- [ ] **Update Remote Service**:
    - [ ] SSH into `dev.sulopuis.to`.
    - [ ] Verify/Update `docker-compose.yml` in the remote project directory.
    - [ ] Restart services with `podman-compose up -d` (or `docker-compose`) to pick up the new `localhost/listmonk-custom:latest` image.
- [ ] **Domain Setup**: Ensure `uutiskirje.sulopuis.to` is correctly pointed and handled by the reverse proxy on the server.

### 📧 Email & Integration
- [ ] **Resend Configuration**: Complete SMTP setup in the dashboard using `RESEND_CONFIG.md` as a reference.
- [ ] **Bounce Handling**: (Optional) Design/Implement a middleware to transform Resend webhooks into listmonk's generic bounce format.

### 🛠 Code Maintenance
- [x] **CalVer Implementation**: Added `cal_version` to API and logs, injected via Docker build.
- [ ] **Refactor `GetTplSubject`**: Move it from `cmd/public.go` to a `utils` package.
- [ ] **Auth Cleanup**:
    - [ ] Review and replace `context.TODO()` in `internal/auth/auth.go`.
    - [ ] Remove legacy token logic as noted in `internal/auth/auth.go:297`.
- [ ] **I18n**: Investigate if "private lists list" should be shown on opt-in emails (`internal/manager/manager.go:381`).
- [ ] **Testing & QA**:
    - [ ] Implement Go unit tests for `internal/core`.
    - [ ] Setup integration tests with `testcontainers-go`.
    - [ ] Add Vitest/Jest for frontend component testing.
    - [ ] Refactor Cypress tests to reduce flakiness and `cy.wait` usage.

---

## Global Tasks (Other Projects)

### 💳 Stripe Transistor Link (vikasietotila repo)
- [ ] **Dockerfile**: Create the `Dockerfile` in the `stripe_transistor_link/` directory.
- [ ] **CI/CD Pipeline**:
    - [ ] Setup GitHub Action for building and pushing to GHCR.
    - [ ] Configure staging deployment.
    - [ ] Implement manual promotion to production.
