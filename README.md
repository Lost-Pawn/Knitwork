# Knitwork

A shared equipment platform for small tailors and self-help groups. Book garment-making equipment, arrange qualified operators, and manage approvals, handovers and returns.

**Stack:** Next.js · TypeScript · Go REST API · PostgreSQL · Docker

## Features

- Resource listings, eligibility checks and operator scheduling.
- Approvals, waiting lists and fair-use quotas.
- Check-in and return photos, incident reporting and activity history.
- Role-based access and in-app notifications.
- English, Hindi, Marathi, Bengali, Tamil and Telugu interface translations.
- Help chat with cited guidance and optional AI-generated answers.

## Run locally

Requires Docker and Docker Compose.

```bash
git clone https://github.com/Lost-Pawn/Knitwork.git
cd Knitwork
docker compose up --build -d
```

Open [localhost:8080](http://localhost:8080). Local demo mode works without external API credentials.

Data is stored in a persistent PostgreSQL volume. Stop the app with `docker compose down`; adding `-v` deletes the database.

### Demo accounts

All accounts use the password **`KnitworkDemo2026!`**.

| Role | Email |
| --- | --- |
| Borrower | `asha@knitwork.test` |
| Trained borrower | `ravi@knitwork.test` |
| Manager | `manager@knitwork.test` |
| Owner | `owner@knitwork.test` |
| Administrator | `admin@knitwork.test` |

These accounts are for local demonstration only.

## Configuration

See [.env.example](.env.example) for available settings. Copy it to `.env` to customise local ports or enable AI features.

Generated help answers and translation of user-written content require `HELP_API_BASE_URL`, `HELP_API_KEY` and `HELP_MODEL`. Without them, cited local help and interface translations remain available.

## Deploy to Render

1. Connect this repository to Render and create a Blueprint using [render.yaml](render.yaml).
2. Set `ADMIN_EMAIL` and `ADMIN_PASSWORD`; optionally supply the three `HELP_*` settings.
3. Deploy, check `/api/health`, then sign in with the administrator account to configure the centre.

The Blueprint provisions a Docker service and persistent PostgreSQL database on paid plans. Production uses `DEMO_SEED=false` and does not include demo accounts. Render supplies the database connection and application origin; set `APP_ORIGIN` for a custom domain.

## Tests

Run backend integration tests with a temporary database:

```bash
docker compose -p knitwork-test -f compose.test.yaml up --abort-on-container-exit --exit-code-from test
docker compose -p knitwork-test -f compose.test.yaml down
```

## Project structure

- `frontend/` — Next.js interface
- `backend/` — Go API and database migrations
- `docs/` — Requirements, API reference, report and demo guide

[Requirements](docs/SRS.md) · [API reference](docs/API.md) · [Project report](docs/REPORT.md) · [Demo guide](docs/DEMO.md) · [Validation](docs/VALIDATION.md)
