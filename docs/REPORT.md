# Knitwork Product and Implementation Report

28 October 2026

## Product overview

Knitwork is a functional prototype for a shared garment-production centre serving small tailoring businesses and self-help groups. It coordinates access to embroidery equipment, sewing workstations and cutting tables through verified membership, operator scheduling, human approval and recorded handovers. The implementation includes a Next.js frontend, a Go REST API, persistent PostgreSQL storage, Docker development and test environments, a Render deployment blueprint and a cited help chatbot.

The central design decision is to complete the resource-sharing lifecycle. A borrower returning equipment does not immediately make it available again: staff confirm the return, and reported damage keeps the equipment on hold until the incident is resolved and an inspection is recorded.

## Problem statement and Indian setting

The hackathon asks teams to unlock underused equipment and spaces while protecting owners, users and institutional property. Its required journey includes listing, discovery, eligibility, requests, approval or waiting lists, check-in, condition reporting, return, incidents and an auditable history.

The selected setting is grounded in an existing Indian operating model. The MSME Development and Facilitation Office in Chennai lists common facility centres and a women’s garment cluster offering computerised embroidery, stitching and training facilities. This supports the plausibility of shared garment equipment. It does not establish that any named centre has idle capacity, that its staff have been interviewed, or that they have agreed to adopt Knitwork. [Official common facility directory](https://msmedi-chennai.gov.in/GARMS_Admin/ClusterTN.aspx).

A proposed first adopter is one centre with equipment already in place and a known member base. That centre can offer browser booking while continuing to accept assisted requests by phone. The expected benefits are clearer availability, recorded access requirements and fewer handover disputes. A real pilot must validate those expectations with workers and centre staff.

## Target users and responsibilities

- Small tailors and self-help groups request equipment, describe their work, follow usage conditions, record collection and return, and report problems.
- Resource owners define availability, rates, training rules, turnaround time and usage limits. They review requests involving their property.
- Centre managers verify membership and training, authorise operators, record assisted requests and coordinate incidents and maintenance.
- Administrators manage account roles and suspension and review escalated disputes.
- Operators are scheduled resources with recorded qualifications and working hours. They do not receive automatic permission from the chatbot.

Public registration creates an unverified borrower. Privileged roles are assigned by an administrator, owners must also be verified before listing equipment, and API permissions are checked independently of what the interface displays.

## Core features implemented

Resource discovery supports text search, category filters, maintenance status, operating hours and reserved periods. Each listing explains fees, deposit terms, training requirements and whether supervised access is permitted. The request form records purpose, time and acceptance of conditions.

The approval workflow checks verified active membership, training, operator qualifications and availability, equipment availability, session length, date limits, turnaround time and weekly quota. The owner or manager can approve, reject with a reason, or place the request on a waiting list. Earlier eligible requests receive priority unless an authorised reviewer records an exception; scheduling and safety constraints remain enforced.

Business-group quotas let staff place several member accounts under one allocation limit. Borrowers cannot reset that quota by editing the business name in their profile. The implementation uses per-resource weekly reserved hours, with Monday boundaries in India Standard Time.

Check-in and return require condition notes, a photograph and a checklist acknowledgement. Image access is restricted to the borrower and authorised staff. Check-in rechecks eligibility, including changes since approval. A manager can record an early handover with a reason; an ordinary borrower can check in within fifteen minutes of the start time.

Extensions require review and cannot silently change an existing reservation. Cancellation releases a reservation and notifies waiting members. Incidents support damage, delay, scheduling conflict and other issues, with resolution notes and administrator escalation. External payment notes record a staff-confirmed reference but do not process or verify money transfers.

The centre overview displays request counts, waitlists, overdue sessions, completed reservations and recorded hours. Persistent in-app notifications cover decisions, reminders, cancellations and incidents. A periodic worker uses deduplication keys so repeated runs do not send the same reminder again.

## Interface language and help

The interface uses short forms, status labels, large action buttons, a resource-card layout and one detailed booking view. Responsive layouts accommodate smaller screens, and dialogs support keyboard focus and Escape. The language selector switches English, Hindi, Marathi, Bengali, Tamil and Telugu interface dictionaries and remembers the selection.

The help section uses a conventional chat interface. PostgreSQL full-text retrieval selects passages from the published help guide. Without API credentials, the application returns cited excerpts. With an OpenAI-compatible Chat Completions provider configured, the retrieved context is sent to the model and a generated answer is displayed with source passages. If that provider fails, local help remains available. This is lexical retrieval, not an embedding or vector-database implementation.

The chatbot does not retrieve private bookings or qualification records and has no action tools. User questions and text explicitly submitted for translation are sent to the configured provider. User-authored text remains in its original form unless a provider-backed translation is requested; generated translations are labelled. Native-speaker review and expanded regional-language retrieval remain necessary before real deployment.

## Architecture and technical depth

The frontend is Next.js 16.4.3 with React 19.3.1 and TypeScript 5.9.3. It exports static assets that the Go 1.27.1 service serves alongside its REST API. This keeps authentication and browser requests on one origin. PostgreSQL 17 stores users, sessions, equipment, operators, bookings, photos, incidents, notifications, help passages and audit events. Redis is not required for this release.

Allocation decisions use a PostgreSQL transaction and an advisory lock. Exclusion constraints independently prevent overlapping active reservations for a machine or an assigned operator. This provides a database safeguard even when two reviewers attempt approvals at the same time. The broad allocation lock favours correctness and simplicity for one centre; a larger installation should evaluate more granular locking and load tests.

Session cookies contain random tokens; only token hashes are stored in the database. Passwords use bcrypt. Production cookies are Secure and HttpOnly with SameSite protection. Mutation requests require JSON and an accepted origin. SQL uses parameters, photo size and types are bounded, and role checks protect private records. The audit API only appends records, although a database administrator could still alter them; the history is not a cryptographic ledger.

Photos are stored in PostgreSQL for straightforward persistence across container recreation. This keeps the prototype independent of ephemeral application storage but increases database size and backup volume. Private object storage would be a more suitable expansion path for a large installation.

## Configuration and deployment

Docker Compose builds the frontend and backend, starts PostgreSQL and waits for health checks. The normal database uses a named volume. A separate test composition uses temporary database storage and runs integration tests without touching local demonstration records. Local mode uses fictional accounts and needs no external API credentials.

Production configuration requires a PostgreSQL connection, production mode, the HTTPS application origin and initial administrator credentials. Render can provide the origin through RENDER_EXTERNAL_URL. The optional help settings are HELP_API_BASE_URL, HELP_API_KEY and HELP_MODEL. They enable generated help and user-content translation. The repository contains .env.example, .env.test, version lockfiles, a Dockerfile and render.yaml. Go dependencies are installed from go.mod and go.sum; requirements.txt is a human-readable toolchain reference.

The Render blueprint declares a Docker web service and managed PostgreSQL with private database access. Its chosen plans incur hosting charges. A local build does not establish that a Render deployment has been created; account configuration, a live deployment and judge credentials must be completed before submitting the public product link. [Render Blueprint reference](https://render.com/docs/blueprint-spec).

## Verification results

The Next.js production build and TypeScript checks passed. Go unit and integration tests passed, including a run with the race detector and static analysis with go vet. The disposable Docker Compose test environment also completed successfully. The six locale dictionaries passed automated checks for matching, non-empty message keys. A multi-stage Docker image was built successfully.

Integration tests cover verified owner listing and its audit entry; the complete supervised sharing journey; anonymous and cross-role access; private photo protection; simultaneous approvals; operator collisions across machines; waiting-list exceptions; shared business quotas; eligibility revocation; conflicting extensions; damaged returns and maintenance release; reminder deduplication; session revocation; help retrieval; and mocked provider generation and failure fallback.

A separate HTTP smoke test exercised the running Docker container from request through supervised approval, photo check-in, photo return and confirmation. It produced a completed booking, two stored condition images and five audit events. Both the app and database containers were then recreated while retaining the named volume. Record counts and content digests were unchanged; authenticated downloads of both images retained identical SHA-256 hashes. The condition images are explicitly labelled synthetic fixtures, and the booking records no real equipment use.

A real external model account and a hosted Render deployment have not been exercised. Visual and interactive browser verification was not completed. Automated API tests establish server behaviour, not every aspect of interface usability or accessibility.

## Strengths and limitations

The main strengths are a focused use case, a complete resource lifecycle, database-backed conflict prevention, explainable human decisions, group-aware fairness, persistent evidence and a local setup that does not require paid APIs. The product can be explained through a single worker’s request and handover rather than a broad marketplace story.

The prototype is limited to one centre, same-day sessions and manually maintained operator shifts. It does not independently verify identity or training, operate machines, provide equipment insurance, process deposits, detect physical damage, send SMS or email, or certify compliance. Password recovery, stronger distributed abuse controls, formal accessibility review, backup drills and native-speaker review remain production tasks. No real user study or adoption evidence is claimed.

Reservation hours and completed sharing cycles can help measure resource use. They are not proof of carbon savings, avoided purchases or improved income. Such impact claims require an agreed baseline and real pilot data.

## Presentation and demonstration

Open with a worker’s practical need: Asha can sew locally but needs one hour of embroidery equipment and a qualified operator. Explain why an empty machine slot alone is insufficient. Show the resource rules, request, operator assignment and approval, then complete the before photo, check-in, return photo and owner inspection.

Use one contrasting failure scenario: another request collides with the reserved operator, or a damaged return prevents the next check-in. Show the reason and audit history. Finish by changing the interface language and asking the help chat how to return damaged equipment, making its citations visible.

A suggested seven-slide structure is problem, target users, complete workflow, live demonstration, technical design, trust and accessibility, and pilot plan. Assign presenters the actual journeys they understand: borrower request, manager review, handover and evidence, or architecture and deployment. Do not invent individual contributions.

## Originality disclosure and submission readiness

The project was implemented with substantial Codex assistance for planning, code generation, testing and documentation. It uses the open-source frameworks and packages recorded in the lockfiles and code-based SVG illustrations. No purchased application template was used. Participants must review third-party licences, complete their actual names and contributions, and disclose AI assistance in the submission.

Before judging, confirm the hosted link, repository access, judge accounts, team details and continued hosting availability. The README supplies setup commands and credentials for the fictional local demonstration; docs/DEMO.md expands the presentation guidance. The requirements specification is supplied separately.
