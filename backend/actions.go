package main

import (
	"net/http"
	"strings"
	"time"
)

func (a *App) bookingAction(w http.ResponseWriter, r *http.Request) {
	var in ActionInput
	if !decode(w, r, &in) {
		return
	}
	if len(in.Reason) > 2000 || len(in.Note) > 2000 {
		fail(w, 400, "invalid_input", "Keep notes under 2000 characters.")
		return
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to update booking.")
		return
	}
	defer tx.Rollback(ctx)
	if allocationLock(ctx, tx) != nil {
		fail(w, 500, "server_error", "Unable to update booking.")
		return
	}
	b, e := loadBooking(ctx, tx, idFrom(r))
	if e != nil {
		fail(w, 404, "not_found", "Booking not found.")
		return
	}
	p, e := loadPolicy(ctx, tx, b.Resource)
	if e != nil {
		fail(w, 404, "not_found", "Equipment not found.")
		return
	}
	u := current(r)
	manager := owns(u, p.Owner)
	if u.ID != b.Borrower && !manager {
		fail(w, 403, "forbidden", "You cannot change this booking.")
		return
	}
	pending := b.Status == "requested" || b.Status == "waitlisted"
	requireManager := func() bool {
		if !manager {
			fail(w, 403, "forbidden", "The resource owner or manager must review this action.")
			return false
		}
		return true
	}
	allocationError := func(err error) {
		msg, ok := allocationMessages[err.Error()]
		if !ok {
			fail(w, 500, "server_error", "Unable to check allocation.")
			return
		}
		fail(w, 409, err.Error(), msg)
	}
	switch in.Action {
	case "approve":
		if !requireManager() {
			return
		}
		if !pending {
			fail(w, 409, "invalid_state", "This request cannot be approved in its current state.")
			return
		}
		if b.Start.Before(time.Now().Add(-time.Minute)) {
			fail(w, 409, "expired_request", "The requested start time has passed. Create a new request.")
			return
		}
		if e = checkAllocation(ctx, tx, p, b, in.OperatorID, in.Override, in.Reason); e != nil {
			allocationError(e)
			return
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET status='approved',operator_id=$1,decision_reason=$2 WHERE id=$3", in.OperatorID, in.Reason, b.ID)
	case "reject", "waitlist":
		if !requireManager() {
			return
		}
		if !pending || len(strings.TrimSpace(in.Reason)) < 3 {
			fail(w, 400, "reason_required", "An open request and a reason are required.")
			return
		}
		status := "rejected"
		if in.Action == "waitlist" {
			status = "waitlisted"
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET status=$1,decision_reason=$2 WHERE id=$3", status, in.Reason, b.ID)
	case "cancel":
		if !pending && b.Status != "approved" {
			fail(w, 409, "invalid_state", "Only requests and uncollected reservations can be cancelled.")
			return
		}
		if len(strings.TrimSpace(in.Reason)) < 3 {
			fail(w, 400, "reason_required", "Explain the cancellation.")
			return
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET status='cancelled',decision_reason=$1,extension_end=NULL WHERE id=$2", in.Reason, b.ID)
		if e == nil {
			_, e = tx.Exec(ctx, `INSERT INTO notifications(user_id,booking_id,kind,message) SELECT borrower_id,id,'queue_available','A reservation was cancelled. Ask the centre to review your waiting request.' FROM bookings WHERE resource_id=$1 AND status='waitlisted' AND starts_at<$3 AND busy_until>$2`, b.Resource, b.Start, b.End)
		}
	case "checkin":
		if b.Status != "approved" {
			fail(w, 409, "invalid_state", "Only approved bookings can be checked in.")
			return
		}
		if p.Maintenance {
			fail(w, 409, "maintenance", allocationMessages["maintenance"])
			return
		}
		if time.Now().After(b.End) {
			fail(w, 409, "expired_request", "The session has ended. Contact the manager.")
			return
		}
		if time.Now().Before(b.Start.Add(-15*time.Minute)) && (!manager || len(in.Reason) < 5) {
			fail(w, 409, "too_early", "Check in within 15 minutes of your session, or ask staff to record an early handover reason.")
			return
		}
		var n int
		_ = tx.QueryRow(ctx, "SELECT count(*) FROM bookings WHERE resource_id=$1 AND id<>$2 AND status IN ('in_use','return_pending')", b.Resource, b.ID).Scan(&n)
		if n > 0 {
			fail(w, 409, "awaiting_return", "The previous user has not completed handover.")
			return
		}
		if code := saveCondition(ctx, tx, u.ID, b.ID, "before", in); code != "" {
			fail(w, 400, code, "Add a valid photo, condition note and completed checklist.")
			return
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET status='in_use',checked_in_at=now() WHERE id=$1", b.ID)
	case "return":
		if b.Status != "in_use" {
			fail(w, 409, "invalid_state", "Only equipment in use can be returned.")
			return
		}
		if code := saveCondition(ctx, tx, u.ID, b.ID, "after", in); code != "" {
			fail(w, 400, code, "Add a valid photo, condition note and completed checklist.")
			return
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET status='return_pending',returned_at=now() WHERE id=$1", b.ID)
		if e == nil && in.Damaged {
			_, e = tx.Exec(ctx, "UPDATE resources SET maintenance=true,maintenance_reason='Damage reported at return' WHERE id=$1", b.Resource)
			if e == nil {
				_, e = tx.Exec(ctx, "INSERT INTO incidents(booking_id,reporter_id,kind,description) VALUES($1,$2,'damage',$3)", b.ID, u.ID, in.Note)
			}
		}
	case "confirm_return":
		if !requireManager() {
			return
		}
		if b.Status != "return_pending" || len(strings.TrimSpace(in.Reason)) < 5 {
			fail(w, 400, "reason_required", "Record your return inspection before confirming.")
			return
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET status='completed',extension_end=NULL WHERE id=$1", b.ID)
	case "extend":
		if b.Status != "approved" && b.Status != "in_use" {
			fail(w, 409, "invalid_state", "Only active bookings can request an extension.")
			return
		}
		if !in.End.After(b.End) || len(in.Reason) < 5 {
			fail(w, 400, "invalid_extension", "Choose a later end time and explain the request.")
			return
		}
		if b.Extension != nil {
			fail(w, 409, "extension_pending", "An extension is already awaiting review.")
			return
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET extension_end=$1,extension_reason=$2 WHERE id=$3", in.End, in.Reason, b.ID)
	case "approve_extension", "reject_extension":
		if !requireManager() {
			return
		}
		if b.Extension == nil || (b.Status != "approved" && b.Status != "in_use") || len(in.Reason) < 3 {
			fail(w, 400, "reason_required", "A pending extension and a review reason are required.")
			return
		}
		if in.Action == "approve_extension" {
			b.End = *b.Extension
			if e = checkAllocation(ctx, tx, p, b, b.Operator, in.Override, in.Reason); e != nil {
				allocationError(e)
				return
			}
			_, e = tx.Exec(ctx, "UPDATE bookings SET ends_at=$1,busy_until=$2,extension_end=NULL,estimate=$3 WHERE id=$4", b.End, b.End.Add(time.Duration(p.Buffer)*time.Minute), estimate(p.Rate, b.Start, b.End), b.ID)
		} else {
			_, e = tx.Exec(ctx, "UPDATE bookings SET extension_end=NULL WHERE id=$1", b.ID)
		}
	case "payment":
		if !requireManager() {
			return
		}
		if len(in.Reason) < 3 {
			fail(w, 400, "reason_required", "Record the external payment reference or note.")
			return
		}
		_, e = tx.Exec(ctx, "UPDATE bookings SET payment_note=$1 WHERE id=$2", in.Reason, b.ID)
	default:
		fail(w, 400, "invalid_action", "Unknown booking action.")
		return
	}
	if e == nil {
		e = audit(ctx, tx, u.ID, &b.ID, &b.Resource, in.Action, strings.TrimSpace(in.Reason+" "+in.Note))
	}
	if e == nil {
		e = notify(ctx, tx, b.Borrower, b.ID, in.Action, "Your booking has an update. Open it to review the details.")
	}
	if e == nil && p.Owner != b.Borrower {
		e = notify(ctx, tx, p.Owner, b.ID, in.Action, "A booking has an update requiring review.")
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 409, "update_conflict", "The change could not be saved. Refresh and check the current booking.")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
