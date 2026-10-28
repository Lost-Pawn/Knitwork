# REST API

All mutation requests use `Content-Type: application/json`. Authentication uses the `knitwork_session` HTTP-only cookie. Browser requests must come from the configured application origin. Errors contain `code` and `error`; the interface uses the code for localisation.

| Method and path | Purpose | Access |
| --- | --- | --- |
| GET /api/health | PostgreSQL readiness | Public |
| GET /api/config | Demo and translation availability | Public |
| POST /api/auth/register | Create unverified borrower | Public |
| POST /api/auth/login | Start 24-hour revocable session | Public |
| POST /api/auth/logout | Revoke session | Session holder |
| GET, PUT /api/me | Profile and verification request | Signed in |
| GET /api/resources | Resource catalogue | Signed in |
| POST /api/resources | List equipment | Owner, manager, admin |
| GET /api/resources/{id}/availability | Reserved intervals without borrower identity | Signed in |
| PATCH /api/resources/{id} | Pause or release after inspection | Resource owner, manager, admin |
| GET /api/operators | Operator catalogue | Owner, manager, admin |
| POST /api/operators | Create or update an operator | Manager, admin |
| GET /api/bookings | Accessible booking records | Borrower sees own; owner sees own resources; staff see centre |
| POST /api/bookings | Request resource access | Borrower; manager can act for another member |
| GET /api/bookings/{id} | Conditions, incidents, audit history | Participants and authorised staff |
| POST /api/bookings/{id}/actions | Review, handover, extension or cancellation | Checked by action and ownership |
| GET /api/conditions/{id}/image | Private image bytes | Participants and authorised staff |
| GET /api/members | Membership review queue | Manager, admin |
| PATCH /api/members/{id} | Verification, training and quota group | Manager, admin; only admin changes roles or suspension |
| POST /api/incidents | Report issue on a booking | Participants and authorised staff |
| PATCH /api/incidents/{id} | Resolve or escalate issue | Appropriate reviewer; escalated resolution requires admin |
| GET /api/notifications | Own notification history | Signed in |
| POST /api/notifications/read | Mark own notifications read | Signed in |
| GET /api/dashboard | Resource utilisation summary | Owner, manager, admin |
| GET /api/help/docs | Published guide | Signed in |
| POST /api/help/chat | Retrieve cited help or generate grounded answer | Signed in |
| POST /api/translate | Labelled provider-backed translation | Signed in |

## Booking example

```json
{
  "resource_id": 1,
  "starts_at": "2026-11-02T04:30:00Z",
  "ends_at": "2026-11-02T05:30:00Z",
  "purpose": "Embroider a small garment batch",
  "assisted": true,
  "accepted": true
}
```

Use future dates. The UI displays India Standard Time and the API stores UTC instants. A successful creation returns `{ "id": 1 }`.

Actions are `approve`, `reject`, `waitlist`, `cancel`, `checkin`, `return`, `confirm_return`, `extend`, `approve_extension`, `reject_extension` and `payment`. Approval can specify `operator_id`, `override` and `reason`. Check-in and return require `note`, `checklist: true` and an `image` data URL under the enforced upload limit. Return can set `damaged: true`. The API never treats an extension request as an approved extension.
