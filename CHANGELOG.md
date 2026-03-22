# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Calendar Versioning](https://calver.org/).

## [26.03.22.2010] - 2026-03-22

### Added
- Started the implementation of Go unit tests for the backend, beginning with `internal/core` utility functions.
- Added a new `Testing & QA` section to the roadmap in `todo.md` to address the lack of backend test coverage.

### Changed
- Analyzed the existing test suite (Cypress E2E) and identified critical weaknesses in backend testing coverage.

## [26.03.18.2008] - 2026-03-18

### Fixed
- Fixed a bug where the editor's main format/template selector toolbar would sometimes become invisible or conflict with the editor's internal sticky tools.

## [26.03.18.2007] - 2026-03-18

### Changed
- Improved the campaign editor UI by making toolbars sticky and implementing internal scrolling for the content area in Markdown, Rich Text, and HTML editors. This ensures tools remain accessible even when writing long content.

## [26.03.18.2006] - 2026-03-18

### Fixed
- Fixed a bug where superadmins (and users with global permissions) saw no media files in the browser due to an incorrect SQL CARDINALITY check against NULL prefix arrays.
- Improved the media isolation logic to correctly handle users with a mix of "get" and "manage" list permissions by returning a union of permitted list IDs.
- Fixed a bug in the filesystem media provider's `GetBlob` that prevented thumbnails and images in isolated subdirectories from being correctly processed and served.
- Fixed the media browser not resetting to the first page after an upload, which sometimes caused newly uploaded files to be hidden on a previous page.

## [26.03.17.2005] - 2026-03-17

### Fixed
- Fixed missing `fmt` and `internal/auth` imports in `cmd/media.go` which caused compilation failures.

## [26.03.17.2004] - 2026-03-17

### Added
- Pre-fetch lists on campaign view mount to populate the "Filter by list" autocomplete immediately.

### Fixed
- Fixed the "Filter by list" and general search functionality in the campaign view and list view by using `PLAINTO_TSQUERY` in the backend, which safely handles search strings containing special characters like `%`.
- Added missing `github.com/lib/pq` import in `internal/core/media.go` to fix compilation error.

## [26.03.16.2001] - 2026-03-16

### Added
- Added media isolation for restricted users. Users with list-specific permissions are now restricted to uploading and viewing media within their own "virtual folders" (e.g., `list-{id}/filename`).
- Updated the filesystem media provider to support subdirectory creation.

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
