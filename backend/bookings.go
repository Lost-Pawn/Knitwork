package main

import (
	"net/http"
	"strings"
	"time"
)

func (a *App) bookings(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	items, e := listJSON(r.Context(), a.db, `SELECT to_jsonb(b)||jsonb_build_object('resource_name',r.name,'borrower_name',u.name,'owner_id',r.owner_id,'operator_name',o.name,'category',r.category,'queue_position',(SELECT count(*) FROM bookings q WHERE q.resource_id=b.resource_id AND q.status='waitlisted' AND(q.created_at,q.id)<=(b.created_at,b.id))) FROM bookings b JOIN resources r ON r.id=b.resource_id JOIN users u ON u.id=b.borrower_id LEFT JOIN operators o ON o.id=b.operator_id WHERE $1 OR b.borrower_id=$2 OR ($3 AND r.owner_id=$2) ORDER BY b.created_at DESC`, staff(u), u.ID, u.Role == "owner")
	if e != nil {
		fail(w, 500, "server_error", "Unable to load bookings.")
		return
	}
	respond(w, 200, items)
}
func (a *App) createBooking(w http.ResponseWriter, r *http.Request) {
	var in BookingInput
	if !decode(w, r, &in) {
		return
	}
	u := current(r)
	if in.BorrowerID == 0 {
		in.BorrowerID = u.ID
	}
	if in.BorrowerID != u.ID && !staff(u) {
		fail(w, 403, "forbidden", "Only a manager can record an assisted request.")
		return
	}
	if !in.Accepted || len(strings.TrimSpace(in.Purpose)) < 5 || len(in.Purpose) > 1000 || in.StartsAt.Before(time.Now().Add(-time.Minute)) {
		fail(w, 400, "invalid_booking", "Accept the conditions, explain your purpose and choose a future time.")
		return
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to request.")
		return
	}
	defer tx.Rollback(ctx)
	if allocationLock(ctx, tx) != nil {
		fail(w, 500, "server_error", "Unable to request.")
		return
	}
	p, e := loadPolicy(ctx, tx, in.ResourceID)
	if e != nil {
		fail(w, 404, "not_found", "Equipment not found.")
		return
	}
	if p.Maintenance {
		fail(w, 409, "maintenance", allocationMessages["maintenance"])
		return
	}
	if e = checkTime(p, in.StartsAt, in.EndsAt); e != nil {
		fail(w, 400, e.Error(), allocationMessages[e.Error()])
		return
	}
	if in.Assisted && p.OperatorPolicy == "none" {
		fail(w, 400, "assistance_unavailable", "This resource does not permit supervised access.")
		return
	}
	var n int
	_ = tx.QueryRow(ctx, `SELECT count(*) FROM bookings WHERE borrower_id=$1 AND resource_id=$2 AND status IN ('requested','waitlisted','approved','in_use','return_pending') AND starts_at<$4 AND busy_until>$3`, in.BorrowerID, in.ResourceID, in.StartsAt, in.EndsAt).Scan(&n)
	if n > 0 {
		fail(w, 409, "duplicate_request", "You already have a request for this period.")
		return
	}
	var id int64
	e = tx.QueryRow(ctx, `INSERT INTO bookings(resource_id,borrower_id,starts_at,ends_at,busy_until,purpose,assisted,estimate) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, in.ResourceID, in.BorrowerID, in.StartsAt, in.EndsAt, in.EndsAt.Add(time.Duration(p.Buffer)*time.Minute), in.Purpose, in.Assisted, estimate(p.Rate, in.StartsAt, in.EndsAt)).Scan(&id)
	if e == nil {
		e = audit(ctx, tx, u.ID, &id, &in.ResourceID, "requested", in.Purpose)
	}
	if e == nil {
		e = notify(ctx, tx, p.Owner, id, "requested", "A booking request needs review.")
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 500, "server_error", "Unable to submit request.")
		return
	}
	respond(w, 201, map[string]int64{"id": id})
}
func (a *App) bookingDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := idFrom(r)
	var borrower, owner int64
	e := a.db.QueryRow(ctx, "SELECT b.borrower_id,r.owner_id FROM bookings b JOIN resources r ON r.id=b.resource_id WHERE b.id=$1", id).Scan(&borrower, &owner)
	if e != nil {
		fail(w, 404, "not_found", "Booking not found.")
		return
	}
	u := current(r)
	if borrower != u.ID && !owns(u, owner) {
		fail(w, 403, "forbidden", "You cannot view this booking.")
		return
	}
	events, e := listJSON(ctx, a.db, `SELECT to_jsonb(a)||jsonb_build_object('actor_name',u.name) FROM audit a LEFT JOIN users u ON u.id=a.actor_id WHERE booking_id=$1 ORDER BY a.id`, id)
	if e != nil {
		fail(w, 500, "server_error", "Unable to load history.")
		return
	}
	conditions, e := listJSON(ctx, a.db, `SELECT to_jsonb(c)-'image'||jsonb_build_object('image_url','/api/conditions/'||c.id||'/image') FROM conditions c WHERE booking_id=$1 ORDER BY id`, id)
	if e != nil {
		fail(w, 500, "server_error", "Unable to load conditions.")
		return
	}
	incidents, e := listJSON(ctx, a.db, `SELECT to_jsonb(i) FROM incidents i WHERE booking_id=$1 ORDER BY id`, id)
	if e != nil {
		fail(w, 500, "server_error", "Unable to load incidents.")
		return
	}
	respond(w, 200, map[string]any{"audit": events, "conditions": conditions, "incidents": incidents})
}
