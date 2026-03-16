# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Calendar Versioning](https://calver.org/).

## [26.03.16.2000] - 2026-03-16

### Fixed
- Fixed the campaign subject prefix not being applied to outgoing emails by ensuring the prefix data is fetched during campaign processing.

## [26.03.16.1999] - 2026-03-16

### Fixed
- Fixed the "Campaign subject prefix" setting for lists not loading correctly in the UI due to camelCase mapping.

## [26.03.16.1998] - 2026-03-16

### Fixed
- Fixed a syntax error and duplicate method in `models/campaigns.go` introduced in the previous version.

## [26.03.16.1997] - 2026-03-16

### Added
- Added list-specific "Campaign subject prefix" setting to automatically prepend a prefix (e.g., "[List name]") to outgoing campaign subjects.
- Added a global fallback "Campaign subject prefix" setting (under Settings -> General) for lists without a specific prefix.

## [26.03.16.1996] - 2026-03-16

### Changed
- Updated CI/CD workflows to use Node.js 24 and resolve Node.js 20 deprecation warnings.
- Updated GitHub Actions to their latest versions (`checkout@v4.2.2`, `setup-go@v5.3.0`, `build-push-action@v6.15.0`, `ssh-action@v1.2.0`).

## [26.03.16.1995] - 2026-03-16

### Fixed
- Fixed Tiptap v3 import issues in the frontend's Richtext editor that caused CI/CD build failures.
- Resolved linting errors in the frontend components.

## [26.03.16.1994] - 2026-03-16

### Added
- Fullscreen mode for the Markdown editor, providing a focused, distraction-free environment for content creation.
- A modern, lightweight Rich Text Editor (Tiptap) as a replacement for TinyMCE, offering a clean toolbar, improved performance, and consistent styling with the Markdown editor.
- Table support, image integration via the media manager, and source code editing within the new Tiptap editor.

### Changed
- Improved the editor UI with a unified toolbar design and unified material-style icons.
- Updated Cypress tests to support the transition to the Tiptap editor.

### Removed
- TinyMCE and its associated dependencies, legacy styles, and language files.

## [26.03.14.1993] - 2026-03-14

### Added
- Implementation of Calendar Versioning (CalVer vYY.MM.DD.N) integrated with the build process and exposed in the API.
- Synchronization with upstream `listmonk/master`.

### Fixed
- Resolved `X-Frame-Options` conflict with Content Security Policy (CSP).
- Explicitly set `X-Frame-Options` to `SAMEORIGIN` for preview endpoints to allow cross-origin previews within the same site.
- Fixed issue where `%` encoded URLs would break when using `TrackLink` (Upstream #2947).
- Fixed attachments incorrectly accumulating for every recipient in test mails (Upstream #2949).
- Missing `uploads` directory in container by adding a persistent volume mount in `infra/registry.json`.

### Changed
- Updated Go version to 1.26.1.
- Implemented build caching in the CI/CD pipeline and Dockerfile for faster builds.
- Optimized Docker multi-stage build order and caching.
- Moved database migrations to run automatically within the Dockerfile `CMD`.
- Restored default configuration loading while maintaining support for auto-migrations.

## [26.03.12.1922] - 2026-03-12

### Fixed
- Fixed incorrect timestamps in dashboard analytics materialized views (Upstream #2952).

## [26.03.08.1920] - 2026-03-08

### Added
- Added expiry and TTL to Altcha CAPTCHA tokens (Upstream #2684).
- Comprehensive documentation for `Resend` configuration (`RESEND_CONFIG.md`).

### Fixed
- Corrected frontend build order in the multi-stage Dockerfile.
- Ensured static directory structure is correctly created for frontend postinstall in Docker.
- Resolved linting errors across the codebase.

### Changed
- Cleaned up deployment debug logs in CI.
- Added store tags to the deployment workflow for better registry compatibility.
