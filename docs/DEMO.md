# Presentation and demonstration guide

## The story

Asha runs a small tailoring business. She has a sewing machine but needs computerised embroidery for a short batch. A nearby shared garment centre owns the equipment and has spare sessions. The centre needs to know who will use it, whether an operator is required, and what happens if it is returned damaged.

Knitwork coordinates that entire relationship. The product grants accountable access to shared equipment; it does not sell tailoring services or automatically operate a machine.

## Suggested slide sequence

1. **Problem:** specialist equipment can be unaffordable to an individual tailor while a shared centre has unused sessions. Explain access, training, scheduling and handover barriers.
2. **Users:** a tailor or self-help group, the resource owner, the centre manager, an authorised operator and an administrator.
3. **Solution:** show the lifecycle from request to inspected return. Explain that both machine and operator must be available.
4. **Live product:** complete one supervised booking, then show a conflicting request and a damage hold.
5. **Engineering:** show separate Next.js and Go folders, PostgreSQL overlap constraints, business-group quotas and append-only application history.
6. **Trust and accessibility:** explain human verification, private photos, six interface languages and manager-assisted requests.
7. **Impact and next steps:** report completed reservations and reuse sessions, then propose a pilot with one actual centre. Avoid invented adoption or environmental savings.

## A practical demonstration order

Open the app in two browser profiles, one for Asha and one for the manager. Prepare two small fictional condition photos before presenting. Use tomorrow's date within the listed hours. The manager's early check-in action requires an explicit reason, so the demo can proceed without changing the computer clock.

- Show a listing and its rules, rates, availability and training policy.
- Request one hour of embroidery with operator assistance.
- Assign Sunita and approve as the manager.
- Show a second request for that machine or the same operator at the same time. Explain the rejection, then place it on the waiting list.
- Record a before photo, check in, add the after photo and return.
- Confirm the return and read the audit timeline.
- On another booking, report a fault and show why the next borrower cannot check in before inspection and release.
- Switch to an Indian language and ask the help chat about returning damaged equipment. Show the citations and distinguish local retrieval from optional model generation.

## Answers to likely questions

**Why would someone use this instead of WhatsApp?** Staff can still receive a request by phone or chat and enter it on the borrower's behalf. Knitwork provides one availability record, checked eligibility, visible allocation reasons and a persistent handover history.

**Why would an owner participate?** The proposed benefits are better use of spare sessions, fee income where applicable and fewer coordination disputes. These are hypotheses to validate with a centre operator, not established adoption results.

**What makes this more than booking?** A reservation combines eligibility, equipment and operator availability, fair allocation and condition-controlled release. The lifecycle continues after the borrower says they are finished.

**Where is the AI?** The help system retrieves approved passages, optionally generates a grounded answer through the configured provider and shows sources. It has no authority over approvals or penalties. The core workflow works without an external model.

**What is the sustainability benefit?** Repeated access and maintenance can improve use of existing equipment. The prototype measures reservation activity and reuse cycles. Carbon or purchase-avoidance claims need separate evidence.

**Is it ready for real production?** It is a functional single-centre prototype. A production pilot still needs staff workflow validation, native-speaker review, backup and recovery procedures, stronger abuse controls, password recovery and operational support.

## Submission checklist

- Deployed judge-accessible link and verified credentials.
- GitHub repository access.
- SRS and implementation report.
- Team names and accurate contributions in TEAM.md.
- AI assistance and third-party components disclosed.
- Known limitations stated.
- Hosting and database remain available throughout judging.
