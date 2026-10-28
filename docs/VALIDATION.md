# Validation record

Verified on 28 October 2026. Application code includes commit `816290d`; subsequent commits update documentation. The final locally built image is `dc9047efd090`.

## Automated checks

- Next.js production build and TypeScript check: passed.
- Go unit and integration tests, including the race detector: passed.
- Go static analysis with `go vet ./...`: passed.
- Disposable PostgreSQL Docker Compose integration environment: passed with exit code 0.
- Locale dictionary coverage: six locales with 145 matching non-empty keys each.
- Multi-stage Docker build: passed; app and PostgreSQL health checks passed.

The integration suite covers verified owner listing, its audit event, the full supervised sharing lifecycle, access control, private photo protection, competing approvals, operator conflicts, queue exceptions, shared-group quotas, revoked eligibility, extensions, damage holds, reminders, session revocation and help provider fallback. Reproduction commands are in the README.

## Running-container and persistence checks

A direct HTTP test against the Docker service completed a fictional request, supervised approval, photo check-in, photo return and manager confirmation. Booking 1 reached `completed`, with two condition records and five audit events. No real equipment was operated. Test images visibly say `SYNTHETIC QA FIXTURE`.

The app and database containers were both recreated with `docker compose up -d --force-recreate --wait`, preserving the named database volume. Before-and-after checks found identical counts and content digests for users (6), resources (3), bookings (1), conditions (2) and audit events (5). After restart, both private images were downloaded through authenticated HTTP requests and retained the same SHA-256 hash:

`7cec95c9a2155bd728ce1128f1cc9c1059336295cc5fc492c8305fbebc0f1208`

The completed booking remained accessible, and `/api/health` returned `{"status":"ok"}`. This verifies container recreation with the volume retained; it does not verify disaster recovery or a backup restore.

## Outstanding verification

- Interactive and visual browser testing was blocked by the local browser-access permission decision. Mobile logout was corrected through source review and passed compilation, but has not been exercised in a browser.
- Live model generation and translation require a selected external provider and credentials. Mocked success and failure paths passed.
- Render deployment, public judge access and production credentials have not been exercised.
- Native-speaker translation review, real-user testing and formal accessibility checks remain outstanding.

The SRS and implementation report were rendered and visually inspected before delivery. They should not be used to imply that every acceptance criterion or production-readiness task has been completed.
