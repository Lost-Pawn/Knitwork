# Knitwork Software Requirements Specification

Version 1.1 — 28 October 2026

## Purpose and scope

Knitwork enables a shared garment production centre to lend access to embroidery machines, sewing workstations and cutting tables to small tailors and self help groups. The product coordinates equipment, authorised operators, approvals, fair allocation, condition handovers and accountable returns. The first release serves one centre and its verified members. It does not certify professional competence, process payments or operate equipment.

## People and permissions

- Borrowers register, maintain a profile, request verification, discover resources, request bookings and extensions, check in, submit return records and report incidents.
- Verified owners list and manage their resources and review associated requests, condition reports and incidents.
- Managers coordinate all centre resources, operators, verification, waiting lists and maintenance.
- Administrators additionally manage account roles and suspension. Public registration always creates an unverified borrower.
- A manager may record an assisted request for a verified borrower; the audit trail identifies both actors.

## Functional requirements

FR01 Authentication shall use hashed passwords and revocable expiring server sessions in secure production cookies. Accounts expose membership verification, training approvals and role. Credentials and uploaded evidence shall not be public.

FR02 Verified owners or authorised centre staff shall list a resource with category, description, location, operating hours, availability date window, maximum session duration, weekly usage cap, training requirements, operator policy, usage conditions, hourly rate and deposit terms. Resources can be paused for maintenance.

FR03 Borrowers shall search and filter resources by text, category and operational availability, inspect access requirements and view occupied periods before requesting a time. Times shall be displayed in the centre timezone with explicit date and time.

FR04 Eligibility shall check verified active membership, required manager approved training and resource conditions. Where owner policy permits assisted use, an authorised operator must be assigned before approval. Missing requirements shall be explained; self declaration never grants qualifications.

FR05 Booking requests shall record borrower, resource, purpose, time range, requested assistance, price estimate and accepted usage conditions. Owners or managers approve, reject with reason or waitlist. Borrowers may cancel before check in. Managers may cancel with a recorded reason.

FR06 Approval and extension shall atomically prevent overlapping active allocations of a resource and any assigned operator, including turnaround time. Availability and weekly usage limits shall be rechecked at approval. Operators shall have active status, qualifications and working hours. Approval conflicts produce a readable reason rather than an inconsistent reservation.

FR07 A transparent queue shall order waitlisted requests by original request time. A per member or staff assigned business group weekly hours cap and earliest eligible waiting request checks shall discourage monopolisation. An owner or manager may override queue order only with a recorded reason; safety constraints cannot be overridden. A cancelled reservation notifies waiting borrowers but does not silently approve them.

FR08 Borrowers or authorised staff shall check in an approved booking with condition notes, checklist acknowledgement and a photograph. The booking must be within its approved session. Only authorised staff may record an early handover with an explicit reason.

FR09 Checkout shall collect a condition note, photograph and damage declaration. It changes the booking to return pending, not completed. The owner or manager confirms return and resource readiness. A reported fault places the resource on maintenance hold pending review.

FR10 Borrowers shall request a revised end time. Authorised reviewers shall accept or reject with a reason after rechecking resource, operator, hours and queue conflicts. Delays shall be visible and notified. Extension requests themselves shall not change an approved slot.

FR11 Participants shall report damage, delay, scheduling conflict or another incident. The responsible owner or manager records review notes and resolution. A disputed incident may be escalated to an administrator. Penalties and liability shall never be decided automatically. Evidence is visible only to authorised participants and staff.

FR12 Persistent in app notifications shall cover decisions, cancellations, queue opportunities, upcoming sessions, overdue handovers and incident updates. A restart safe periodic worker shall create deduplicated reminders. Users mark their notifications as read.

FR13 Every material action shall append an audit event containing actor, entity, action, timestamp and safe explanatory details. Relevant history shall be visible to participants and authorised staff. The application shall not provide audit editing or deletion endpoints.

FR14 The interface shall prioritise short labels, large controls, visible next steps, responsive layouts, keyboard access, clear empty states and text explanations alongside status colours. It shall provide complete interface dictionaries in English, Hindi, Marathi, Bengali, Tamil and Telugu, with persisted language preference. User supplied content remains attributable to its original author; optional credential backed translation shall provide labelled translations while preserving originals.

FR15 A classic help chat shall retrieve centre approved help passages and show source citations. Without external credentials it shall return useful cited excerpts from the local help corpus. With an OpenAI compatible chat completion provider configured it shall generate grounded answers from retrieved passages. Unsupported questions shall lead to a manager contact instruction. The assistant cannot approve access, alter bookings or decide penalties. Private user documents are excluded from retrieval.

FR16 A manager dashboard shall show requests needing action, waitlists, overdue sessions, maintenance holds and resource utilisation. These figures represent recorded usage, not claimed carbon savings or independently verified social impact.

## Data and architecture

The frontend shall use Next.js with TypeScript and static export. The backend shall use Go and a REST JSON API, serving the exported frontend from the same origin. PostgreSQL is the system of record for users, sessions, resources, operators, bookings, extension requests, condition photographs, incidents, notifications, help documents and audit events. Keeping bounded image data in PostgreSQL makes persistence independent of container filesystems and simplifies a one service Render deployment. Redis is unnecessary for this release.

Database migrations and optional fictional demo seeding shall run deterministically on startup. Production seeding shall be disabled by default. PostgreSQL constraints and transactional locking shall protect scheduling correctness. SQL shall be parameterised. Uploads shall have enforced type and size limits. API responses shall not expose passwords, session tokens or private evidence to unrelated users.

## Deployment and configuration

DEP01 A multi stage Dockerfile shall build frontend assets and the Go executable. Docker Compose shall start the application and PostgreSQL with health checks and a named database volume. A separate disposable test composition shall run integration checks without touching local development data.

DEP02 The repository shall contain .env.example and an explicitly fictional local .env.test. Local Compose shall work with documented safe demo credentials and no third party API key. Production configuration shall require DATABASE_URL, APP_ORIGIN and production mode, with optional help model and translation provider credentials. No genuine credentials shall be committed.

DEP03 A Render Blueprint shall configure a Docker web service and persistent PostgreSQL database. The app shall bind PORT, use production cookies and expose a readiness endpoint. Actual hosted deployment requires the user's Render account and plan selection; the implementation must not imply an unperformed deployment succeeded.

DEP04 Dependency versions shall be locked in go.mod, go.sum, package.json and package-lock.json. requirements.txt shall document the pinned toolchain and dependency installation commands; Go does not use pip requirements files.

## Acceptance and verification

AT01 A judge can log in as a manager, list a resource, verify a borrower, approve a request, record check in, receive a return and inspect the audit history.

AT02 Two simultaneous approvals for overlapping machine or operator slots cannot both succeed. Weekly caps and queue order produce explainable decisions.

AT03 An unverified borrower cannot receive approval. An unrelated borrower cannot inspect private booking evidence. Public signup cannot grant privileged roles.

AT04 A damaged return holds the resource until authorised review and repair release. An extension that conflicts with an allocation fails without changing the original booking.

AT05 Cancellation, reminders, overdue processing and queue updates persist without duplicate notifications after worker restart.

AT06 Data and photographs survive application container recreation. Fresh local Compose works without external credentials. The test composition exercises a separate database.

AT07 Every supported locale covers the complete application message dictionary. The help chat returns citations, refuses unsupported answers and works in local retrieval mode when an external provider is unavailable.

AT08 Production assets compile, Go unit and integration tests pass, browser journeys are checked and the Docker image builds. Validation results and untested external integrations shall be disclosed accurately.

## Boundaries and future work

The release does not perform Aadhaar verification, payment settlement, physical access control, legal certification, automatic liability assignment or live sensor monitoring. Email and SMS, multiple centres, SSO and independent document verification are future integrations. Local demo identities are fictional. AI assistance, frameworks and existing components shall be disclosed in the README and submission report. Team names and actual contributions must be completed by the participants before submission.
