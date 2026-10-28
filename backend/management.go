package main

import (
	"fmt"
	"net/http"
	"strings"
)

func (a *App) members(w http.ResponseWriter, r *http.Request) {
	if !staff(current(r)) {
		fail(w, 403, "forbidden", "Only managers can review membership.")
		return
	}
	v, e := listJSON(r.Context(), a.db, "SELECT to_jsonb(u)-'password_hash' FROM users u ORDER BY id")
	if e != nil {
		fail(w, 500, "server_error", "Unable to load members.")
		return
	}
	respond(w, 200, v)
}
func (a *App) verifyMember(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !staff(u) {
		fail(w, 403, "forbidden", "Only managers can review membership.")
		return
	}
	var in struct {
		AllocationGroup string   `json:"allocation_group"`
		Verified        bool     `json:"verified"`
		Qualifications  []string `json:"qualifications"`
		Reason          string   `json:"reason"`
		Role            string   `json:"role"`
		Suspended       *bool    `json:"suspended"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(strings.TrimSpace(in.Reason)) < 5 {
		fail(w, 400, "reason_required", "Record how membership and training were checked.")
		return
	}
	if len(in.AllocationGroup) > 100 {
		fail(w, 400, "invalid_input", "Keep allocation groups under 100 characters.")
		return
	}
	if in.Qualifications == nil {
		in.Qualifications = []string{}
	}
	if (in.Role != "" || in.Suspended != nil) && u.Role != "admin" {
		fail(w, 403, "forbidden", "Only administrators can change roles or suspend accounts.")
		return
	}
	if in.Role != "" && in.Role != "borrower" && in.Role != "owner" && in.Role != "manager" && in.Role != "admin" {
		fail(w, 400, "invalid_role", "Invalid role.")
		return
	}
	if idFrom(r) == u.ID && (in.Role != "" || (in.Suspended != nil && *in.Suspended)) {
		fail(w, 400, "self_change", "Another administrator must change your access.")
		return
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to save.")
		return
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, "UPDATE users SET verified=$1,qualifications=$2,role=COALESCE(NULLIF($3,''),role),suspended=COALESCE($4,suspended)  ,allocation_group=$6 WHERE id=$5", in.Verified, in.Qualifications, in.Role, in.Suspended, idFrom(r), strings.ToLower(strings.TrimSpace(in.AllocationGroup)))
	if e == nil {
		e = audit(ctx, tx, u.ID, nil, nil, "member_reviewed", fmt.Sprintf("Member %d: %s", idFrom(r), in.Reason))
	}
	if e == nil {
		_, e = tx.Exec(ctx, "INSERT INTO notifications(user_id,kind,message) VALUES($1,'verification','Your membership or training verification was updated.')", idFrom(r))
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 500, "server_error", "Unable to save member review.")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) createIncident(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BookingID   int64  `json:"booking_id"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Description) < 5 || len(in.Description) > 2000 || !strings.Contains("|damage|delay|conflict|other|", "|"+in.Kind+"|") {
		fail(w, 400, "invalid_incident", "Choose a category and describe what happened.")
		return
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to report.")
		return
	}
	defer tx.Rollback(ctx)
	if allocationLock(ctx, tx) != nil {
		fail(w, 500, "server_error", "Unable to report.")
		return
	}
	b, e := loadBooking(ctx, tx, in.BookingID)
	if e != nil {
		fail(w, 404, "not_found", "Booking not found.")
		return
	}
	p, e := loadPolicy(ctx, tx, b.Resource)
	if e != nil {
		fail(w, 404, "not_found", "Resource not found.")
		return
	}
	u := current(r)
	if u.ID != b.Borrower && !owns(u, p.Owner) {
		fail(w, 403, "forbidden", "You cannot report on this booking.")
		return
	}
	var id int64
	e = tx.QueryRow(ctx, "INSERT INTO incidents(booking_id,reporter_id,kind,description) VALUES($1,$2,$3,$4) RETURNING id", b.ID, u.ID, in.Kind, in.Description).Scan(&id)
	if e == nil && in.Kind == "damage" {
		_, e = tx.Exec(ctx, "UPDATE resources SET maintenance=true,maintenance_reason='Incident awaiting review' WHERE id=$1", b.Resource)
	}
	if e == nil {
		e = audit(ctx, tx, u.ID, &b.ID, &b.Resource, "incident_reported", in.Description)
	}
	if e == nil {
		e = notify(ctx, tx, p.Owner, b.ID, "incident", "An incident needs review.")
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 500, "server_error", "Unable to save incident.")
		return
	}
	respond(w, 201, map[string]int64{"id": id})
}
func (a *App) resolveIncident(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status     string `json:"status"`
		Resolution string `json:"resolution"`
	}
	if !decode(w, r, &in) {
		return
	}
	if (in.Status != "resolved" && in.Status != "escalated") || len(in.Resolution) < 5 {
		fail(w, 400, "reason_required", "Record a resolution or escalation reason.")
		return
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to update.")
		return
	}
	defer tx.Rollback(ctx)
	var booking, resource, owner, borrower int64
	var status string
	e = tx.QueryRow(ctx, `SELECT i.booking_id,b.resource_id,r.owner_id,b.borrower_id,i.status FROM incidents i JOIN bookings b ON b.id=i.booking_id JOIN resources r ON r.id=b.resource_id WHERE i.id=$1 FOR UPDATE OF i`, idFrom(r)).Scan(&booking, &resource, &owner, &borrower, &status)
	u := current(r)
	if e != nil {
		fail(w, 404, "not_found", "Incident not found.")
		return
	}
	if in.Status == "resolved" && (!owns(u, owner) || (status == "escalated" && u.Role != "admin")) {
		fail(w, 403, "forbidden", "An authorised reviewer must resolve this incident.")
		return
	}
	if in.Status == "escalated" && u.ID != borrower && !owns(u, owner) {
		fail(w, 403, "forbidden", "You cannot escalate this incident.")
		return
	}
	_, e = tx.Exec(ctx, "UPDATE incidents SET status=$1,resolution=$2 WHERE id=$3", in.Status, in.Resolution, idFrom(r))
	if e == nil {
		e = audit(ctx, tx, u.ID, &booking, &resource, "incident_"+in.Status, in.Resolution)
	}
	if e == nil {
		e = notify(ctx, tx, borrower, booking, "incident", "The incident review was updated.")
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 500, "server_error", "Unable to update incident.")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) notifications(w http.ResponseWriter, r *http.Request) {
	v, e := listJSON(r.Context(), a.db, "SELECT to_jsonb(n) FROM notifications n WHERE user_id=$1 ORDER BY id DESC LIMIT 100", current(r).ID)
	if e != nil {
		fail(w, 500, "server_error", "Unable to load notifications.")
		return
	}
	respond(w, 200, v)
}
func (a *App) readNotifications(w http.ResponseWriter, r *http.Request) {
	_, e := a.db.Exec(r.Context(), "UPDATE notifications SET read=true WHERE user_id=$1", current(r).ID)
	if e != nil {
		fail(w, 500, "server_error", "Unable to update notifications.")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !staff(u) && u.Role != "owner" {
		fail(w, 403, "forbidden", "This view is for centre staff.")
		return
	}
	v, e := listJSON(r.Context(), a.db, `SELECT jsonb_build_object('id',r.id,'name',r.name,'maintenance',r.maintenance,'requests',count(b.id) FILTER(WHERE b.status='requested'),'waiting',count(b.id) FILTER(WHERE b.status='waitlisted'),'overdue',count(b.id) FILTER(WHERE b.status='in_use' AND b.ends_at<now()),'completed',count(b.id) FILTER(WHERE b.status='completed'),'hours',COALESCE(sum(extract(epoch FROM(b.ends_at-b.starts_at))/3600) FILTER(WHERE b.status='completed'),0)) FROM resources r LEFT JOIN bookings b ON b.resource_id=r.id WHERE $1 OR r.owner_id=$2 GROUP BY r.id ORDER BY r.id`, staff(u), u.ID)
	if e != nil {
		fail(w, 500, "server_error", "Unable to load centre summary.")
		return
	}
	respond(w, 200, v)
}
