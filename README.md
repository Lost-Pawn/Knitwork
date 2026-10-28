# Knitwork

**Shared equipment and skilled support for small garment makers.**

Knitwork helps a garment-production centre give small tailors and self-help groups access to embroidery equipment, sewing workstations and cutting tables. Owners retain control over eligibility, operators, approvals, handovers and repairs.

The application implements a complete sharing journey:

**List → request → verify → approve or waitlist → check in → return → inspect → release.**

## Run locally

Install Docker with Docker Compose, then:

```bash
git clone https://github.com/Lost-Pawn/Knitwork.git
cd Knitwork
docker compose up --build -d
```

Open **http://localhost:8080**. The first build downloads pinned base images and dependencies. Later starts reuse the image. No paid API, local Go installation or local Node installation is required for this path.

```bash
docker compose ps
docker compose logs -f app
docker compose down
```

The PostgreSQL named volume retains accounts, bookings, audit history and photographs when containers are stopped or recreated. `docker compose down -v` deletes the local database; do not use it if you want to keep your data.

If a port is occupied, copy `.env.example` to `.env`, change `APP_PORT` or `DB_PORT`, and start Compose again. Local services bind to loopback, not your entire network. `.env.test` contains fictional settings for a separate local demonstration; run `docker compose -p knitwork-demo-test --env-file .env.test up --build -d` to start that second demo on port 8082 with its own named volume.

## Fictional demo accounts

Every local demo account uses **`KnitworkDemo2026!`**.

| Account | Email | Purpose |
| --- | --- | --- |
| Asha | `asha@knitwork.test` | Verified member; sewing training, supervised embroidery access |
| Ravi | `ravi@knitwork.test` | Verified member; sewing and embroidery training |
| Manager | `manager@knitwork.test` | Reviews members, equipment, operators and bookings |
| Owner | `owner@knitwork.test` | Lists and manages their own equipment |
| Administrator | `admin@knitwork.test` | Manages roles, suspension and escalated incidents |

Demo login buttons appear only when `DEMO_SEED=true` outside production. New public signups are always unverified borrowers. Demo data does not represent real people or an affiliated centre.

## What works

- Searchable equipment listings with rules, dates, operating hours, fees, deposits, training requirements and maintenance status.
- Borrower, owner, manager and administrator permissions enforced by the Go API.
- Human membership and training review, plus manager-entered requests for members who prefer assisted booking.
- Coordinated equipment and operator scheduling, with turnaround buffers.
- PostgreSQL exclusion constraints and transactional allocation checks that prevent overlapping approvals.
- Weekly quotas for an individual account or a staff-assigned business group, transparent waiting order and recorded queue exceptions.
- Approval, rejection, cancellation and reviewed extensions with explanations.
- Check-in and return photographs, checklist acknowledgements and private condition history.
- Owner-confirmed returns, damage holds, incident resolution and administrator escalation.
- Persistent in-app notifications, reminders and overdue alerts.
- A manager overview of requests, completed reservations, waiting members and maintenance.
- English, Hindi, Marathi, Bengali, Tamil and Telugu interface dictionaries.
- A classic help chat that retrieves published guidance with citations, with optional model generation and content translation.

## A five-minute demo

1. Sign in as **Manager**, open **Explore equipment**, and list a resource. For a quick existing example, use **Embroidery studio**.
2. Sign in as **Asha**, select equipment and choose a future slot within its opening hours. Describe a garment batch, request operator assistance and accept the conditions.
3. Sign in as **Manager**, open the request, assign **Sunita More**, add a review reason and approve.
4. Make a competing request with Ravi to show a conflict or waiting-list decision. Show why operator availability matters independently of machine availability.
5. For an immediate presentation, the manager can check in the approved future booking with an explicit early-handover reason. Upload a small JPEG, PNG or WebP and confirm the checklist. Ordinary borrower check-in opens 15 minutes before the session.
6. Asha records the final condition and returns the equipment. The manager confirms the inspection. Show the transaction timeline and both photographs.
7. On a second booking, report a fault. Show the maintenance hold, resolve the incident, and record the inspection before releasing the resource.
8. Change the interface language and ask Help: **“How do I return damaged equipment?”** Show its cited sources.

Use two browser profiles or a private window for different roles. All dates and times are **Asia/Kolkata**. A reservation is not permission to bypass on-site safety procedures.

## Architecture

```text
frontend/              Next.js and TypeScript, exported to static assets
backend/               Go REST API, authentication, scheduling and help
backend/migrations/    Versioned PostgreSQL schema
scripts/               Locale validation and reproducible checks
docs/                  SRS, report, API notes and submission checklist
Dockerfile             Multi-stage frontend and Go build
compose.yaml           Local app and persistent PostgreSQL
compose.test.yaml      Disposable PostgreSQL and integration-test runner
render.yaml            Render Docker service and managed PostgreSQL
.env.example           Configuration reference, no real credentials
.env.test              Explicitly fictional local test settings
requirements.txt       Toolchain and dependency reference
```

The Go server serves both `/api/*` and the compiled frontend. A single origin avoids a separate CORS and cross-domain cookie setup. PostgreSQL stores bounded condition photographs alongside transactional data, so uploads do not depend on an ephemeral container filesystem. At larger scale, move photos to private object storage with signed URLs.

Redis is intentionally unnecessary for this release. PostgreSQL supplies persistence, full-text help retrieval, quota queries and allocation locking. The small in-process abuse limiter resets when an instance restarts; a production multi-instance installation should move rate limiting to shared infrastructure.

## Configuration

| Variable | Local value or requirement | Production meaning |
| --- | --- | --- |
| `DATABASE_URL` | Compose sets it automatically | Managed PostgreSQL internal connection string |
| `APP_ENV` | `development` | Must be `production` |
| `APP_ORIGIN` | `http://localhost:8080` | Exact HTTPS origin; on Render the app can use `RENDER_EXTERNAL_URL` automatically |
| `PORT` | `8080` | Render-provided listening port is honoured |
| `DEMO_SEED` | `true` | Must be `false` |
| `ADMIN_EMAIL` | Optional | Initial administrator email |
| `ADMIN_PASSWORD` | Optional | Initial password, 12–72 characters; set a unique secret |
| `STATIC_DIR` | `../frontend/out` when running Go directly | Docker sets `/app/public` |
| `APP_PORT`, `DB_PORT` | `8080`, `5434` | Local Compose port mappings only |
| `POSTGRES_PASSWORD` | Fictional local default | Local Compose only; Render manages its database credentials |
| `HELP_API_BASE_URL` | Empty works | Optional HTTPS OpenAI-compatible API base, usually ending `/v1` |
| `HELP_API_KEY` | Empty works | Optional provider credential |
| `HELP_MODEL` | Empty works | Provider-supported chat model identifier |

**Bootstrap behaviour:** the initial administrator is created only if its email does not exist. Changing `ADMIN_PASSWORD` does not reset an existing account. Supply both admin fields when deploying a fresh production database. The project does not yet include password reset or email verification.

Never commit `.env` or real credentials. If a database password contains URL-special characters, URL-encode it when constructing a `DATABASE_URL` manually.

## Help chatbot and translation

The local mode searches centre help documents in PostgreSQL and returns ranked excerpts with citations. This is an extractive fallback, **not a claim that an LLM runs locally**. When all three `HELP_*` settings are configured, the backend retrieves passages and calls the provider's `/chat/completions` endpoint to generate a grounded answer. A provider failure falls back to the published excerpts.

The knowledge base is the public centre guide in `backend/help.go`; no borrower records or private photographs are indexed. The assistant has no action tools and cannot approve access, modify reservations or decide penalties. The implementation uses PostgreSQL lexical retrieval rather than embeddings or a vector database. Live provider behaviour still needs verification with your selected account and model.

All interface dictionaries work without external credentials. User-authored descriptions, notes, help passages and error details retain their original content. With a configured provider, the **Translate this text** action supplies a labelled translation without overwriting the source. User questions and text explicitly submitted for translation are sent to that configured provider. Names and identifiers are preserved. Regional-language help retrieval uses a small keyword mapping and should be expanded and reviewed with native speakers before a real pilot.

## Deploy to Render

1. Push the repository to GitHub and connect it to Render.
2. Create a **Blueprint** from `render.yaml`. It defines one Docker web service and one managed PostgreSQL database. The declared compute and storage plans are paid; review Render's current costs before creating them.
3. Fill in `ADMIN_EMAIL` and `ADMIN_PASSWORD`. Leave the help settings empty for local-guide mode, or provide all three for generated answers and translation.
4. The Blueprint connects `DATABASE_URL` automatically. The app reads Render's HTTPS `RENDER_EXTERNAL_URL`; set `APP_ORIGIN` explicitly if you use a custom domain.
5. Deploy and wait for `/api/health` to return `{"status":"ok"}`.
6. Sign in with the bootstrap administrator, add equipment and operators, then register and verify the judge accounts you intend to share. Production does not contain the local demo accounts.
7. Keep the service and database available for the judging period. Provide the deployed link and judge credentials in your submission.

The database allows private network connections through the Blueprint. There is no dependency on a Render persistent disk. The app runs as a non-root container user and uses `Secure`, `HttpOnly`, `SameSite=Lax` cookies in production.

References: [Render Blueprints](https://render.com/docs/blueprint-spec), [compute plans](https://render.com/docs/compute-plans), [PostgreSQL connections](https://render.com/docs/postgresql-creating-connecting).

A successful local image build is not proof of a hosted deployment. Deploying to a Render account is a separate step.

## Tests

Run the entire API/database integration suite in an isolated, temporary database:

```bash
docker compose -p knitwork-test -f compose.test.yaml up --abort-on-container-exit --exit-code-from test
docker compose -p knitwork-test -f compose.test.yaml down
```

This test environment uses temporary PostgreSQL storage and does not connect to the regular local database. The integration runner refuses a database URL without the `knitwork_test` database name.

For frontend checks:

```bash
cd frontend
npm ci
npm run build
cd ..
node scripts/check-locales.mjs
```

For Go development, use the version pinned in `backend/go.mod`:

```bash
cd backend
go mod download
go test ./...
go vet ./...
```

Integration tests are skipped by `go test` unless `TEST_DATABASE_URL` points to a disposable `knitwork_test` database. With that URL configured, `go test -race -v ./...` also checks data races. The suite covers the full sharing journey, authorisation, concurrent reservations, operator collisions, group quotas, queue exceptions, revoked eligibility, extensions, damage holds, reminder deduplication and a mocked model provider.

## Development without rebuilding Docker

Start only PostgreSQL, install dependencies, build the frontend, then run Go:

```bash
docker compose up -d db
cd frontend
npm ci
npm run build
cd ../backend
go mod download
DATABASE_URL='postgres://knitwork:local-knitwork-only@localhost:5434/knitwork?sslmode=disable' APP_ORIGIN=http://localhost:8080 DEMO_SEED=true go run .
```

Use `go.mod` and `go.sum` for Go dependencies, and `package.json` plus `package-lock.json` for frontend dependencies. `requirements.txt` is a readable reference; it is not a Python installation file.

## Known limitations and next steps

- One centre, a small resource catalogue, same-day sessions and manually maintained operator shifts. There are no recurring bookings or suggested alternative slots.
- Human verification is recorded, not independently performed through a government or qualification registry.
- Physical inspection, cleaning and training remain staff responsibilities. The application cannot verify the truth of a checklist or photo.
- Payments, refundable deposits and disputes are recorded for human review; no payment settlement or automatic penalties are implemented.
- The audit trail is append-only through the application API, but database administrators can still alter database records. It is not a cryptographically tamper-proof ledger.
- The UI is responsive and keyboard-oriented, but accessibility certification and native-speaker review are outside this prototype.
- User-supplied content requires provider-backed translation; fixed interface text is translated locally.
- Notifications are in-app. Email, SMS, push notifications, recovery flows, staff calendar integrations and multi-centre support are future work.
- Measured reservation hours are operational metrics, not verified carbon savings or proof of adoption. Interview a real centre operator before making adoption claims.

## Original work and disclosure

This project was implemented with substantial AI assistance using Codex for design discussion, code generation, testing and documentation. It uses Go, pgx, x/crypto, Next.js, React, TypeScript, Docker, PostgreSQL and other packages recorded in lockfiles. The interface illustrations are SVG code. No purchased project or existing application template was used.

Review package licences before distribution. Record each participant's actual work in `docs/TEAM.md`; the generated history does not establish individual team contributions. Follow the hackathon's official build-window and disclosure requirements.

## Project documents

- [Requirements specification](docs/SRS.md) · [Download Word document](docs/Knitwork-SRS.docx)
- [Product and implementation report](docs/REPORT.md) · [Download Word document](docs/Knitwork-Report.docx)
- [API reference](docs/API.md)
- [Presentation and demo guide](docs/DEMO.md)
- [Verification results and outstanding checks](docs/VALIDATION.md)
- [Team details to complete](docs/TEAM.md)
