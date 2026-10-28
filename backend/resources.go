package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func idFrom(r *http.Request) int64 { id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64); return id }
func (a *App) resources(w http.ResponseWriter, r *http.Request) {
	items, e := listJSON(r.Context(), a.db, `SELECT to_jsonb(r)||jsonb_build_object('owner_name',u.name) FROM resources r JOIN users u ON u.id=r.owner_id ORDER BY r.id`)
	if e != nil {
		fail(w, 500, "server_error", "Unable to load resources.")
		return
	}
	respond(w, 200, items)
}
func validResource(in ResourceInput) bool {
	_, e1 := time.Parse("2006-01-02", in.AvailableFrom)
	_, e2 := time.Parse("2006-01-02", in.AvailableTo)
	return len(strings.TrimSpace(in.Name)) >= 3 && len(in.Name) <= 120 && len(in.Description) <= 2000 && len(in.Location) > 0 && len(in.Rules) > 0 && len(in.Rules) <= 2000 && in.MaxHours >= 1 && in.MaxHours <= 12 && in.WeeklyHours >= 1 && in.WeeklyHours <= 60 && in.OpenHour >= 0 && in.CloseHour <= 24 && in.CloseHour > in.OpenHour && in.BufferMinutes >= 0 && in.BufferMinutes <= 180 && in.HourlyRate >= 0 && in.Deposit >= 0 && e1 == nil && e2 == nil && in.AvailableTo >= in.AvailableFrom && (in.OperatorPolicy == "none" || in.OperatorPolicy == "optional" || in.OperatorPolicy == "required") && (in.Category == "embroidery" || in.Category == "sewing" || in.Category == "cutting")
}
func (a *App) createResource(w http.ResponseWriter, r *http.Request) {
	u := current(r)
	if !staff(u) && u.Role != "owner" {
		fail(w, 403, "forbidden", "Only owners or centre staff can list equipment.")
		return
	}
	var in ResourceInput
	if !decode(w, r, &in) {
		return
	}
	if !validResource(in) {
		fail(w, 400, "invalid_resource", "Check resource details, hours and availability dates.")
		return
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to save.")
		return
	}
	defer tx.Rollback(ctx)
	var id int64
	e = tx.QueryRow(ctx, `INSERT INTO resources(owner_id,name,category,description,location,rules,training,operator_policy,hourly_rate,deposit,max_hours,weekly_hours,open_hour,close_hour,buffer_minutes,available_from,available_to) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) RETURNING id`, u.ID, in.Name, in.Category, in.Description, in.Location, in.Rules, in.Training, in.OperatorPolicy, in.HourlyRate, in.Deposit, in.MaxHours, in.WeeklyHours, in.OpenHour, in.CloseHour, in.BufferMinutes, in.AvailableFrom, in.AvailableTo).Scan(&id)
	if e == nil {
		e = audit(ctx, tx, u.ID, nil, &id, "resource_listed", in.Name)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 500, "server_error", "Unable to save resource.")
		return
	}
	respond(w, 201, map[string]int64{"id": id})
}
func (a *App) availability(w http.ResponseWriter, r *http.Request) {
	items, e := listJSON(r.Context(), a.db, `SELECT jsonb_build_object('starts_at',starts_at,'ends_at',busy_until,'status',status) FROM bookings WHERE resource_id=$1 AND status IN ('approved','in_use','return_pending') AND busy_until>now()-interval '1 day' ORDER BY starts_at`, idFrom(r))
	if e != nil {
		fail(w, 500, "server_error", "Unable to load availability.")
		return
	}
	respond(w, 200, items)
}
func (a *App) maintenance(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Maintenance bool   `json:"maintenance"`
		Reason      string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Reason) == "" {
		fail(w, 400, "reason_required", "Record the inspection or maintenance reason.")
		return
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to save.")
		return
	}
	defer tx.Rollback(ctx)
	id := idFrom(r)
	var owner int64
	e = tx.QueryRow(ctx, "SELECT owner_id FROM resources WHERE id=$1 FOR UPDATE", id).Scan(&owner)
	if e != nil || !owns(current(r), owner) {
		fail(w, 403, "forbidden", "You cannot manage this resource.")
		return
	}
	if !in.Maintenance {
		var n int
		_ = tx.QueryRow(ctx, `SELECT count(*) FROM incidents i JOIN bookings b ON b.id=i.booking_id WHERE b.resource_id=$1 AND i.kind='damage' AND i.status<>'resolved'`, id).Scan(&n)
		if n > 0 {
			fail(w, 409, "open_damage", "Resolve open damage reports before releasing the equipment.")
			return
		}
	}
	_, e = tx.Exec(ctx, "UPDATE resources SET maintenance=$1,maintenance_reason=$2 WHERE id=$3", in.Maintenance, in.Reason, id)
	if e == nil {
		e = audit(ctx, tx, current(r).ID, nil, &id, "resource_readiness_changed", in.Reason)
	}
	if e == nil && in.Maintenance {
		_, e = tx.Exec(ctx, `INSERT INTO notifications(user_id,booking_id,kind,message) SELECT borrower_id,id,'maintenance','Your resource is on a maintenance hold. Contact the centre before travelling.' FROM bookings WHERE resource_id=$1 AND status IN ('approved','requested','waitlisted')`, id)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 500, "server_error", "Unable to save.")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) operators(w http.ResponseWriter, r *http.Request) {
	if !staff(current(r)) && current(r).Role != "owner" {
		fail(w, 403, "forbidden", "Only centre staff can view operator scheduling.")
		return
	}
	items, e := listJSON(r.Context(), a.db, "SELECT to_jsonb(o) FROM operators o ORDER BY id")
	if e != nil {
		fail(w, 500, "server_error", "Unable to load operators.")
		return
	}
	respond(w, 200, items)
}
func (a *App) saveOperator(w http.ResponseWriter, r *http.Request) {
	if !staff(current(r)) {
		fail(w, 403, "forbidden", "Only managers can authorise operators.")
		return
	}
	var in struct {
		ID             int64    `json:"id"`
		Name           string   `json:"name"`
		Qualifications []string `json:"qualifications"`
		Active         bool     `json:"active"`
		OpenHour       int      `json:"open_hour"`
		CloseHour      int      `json:"close_hour"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Name) < 2 || in.OpenHour < 0 || in.CloseHour > 24 || in.CloseHour <= in.OpenHour {
		fail(w, 400, "invalid_input", "Check operator details.")
		return
	}
	if in.Qualifications == nil {
		in.Qualifications = []string{}
	}
	ctx := r.Context()
	tx, e := a.db.Begin(ctx)
	if e != nil {
		fail(w, 500, "server_error", "Unable to save.")
		return
	}
	defer tx.Rollback(ctx)
	if in.ID == 0 {
		e = tx.QueryRow(ctx, "INSERT INTO operators(name,qualifications,active,open_hour,close_hour) VALUES($1,$2,$3,$4,$5) RETURNING id", in.Name, in.Qualifications, in.Active, in.OpenHour, in.CloseHour).Scan(&in.ID)
	} else {
		_, e = tx.Exec(ctx, "UPDATE operators SET name=$1,qualifications=$2,active=$3,open_hour=$4,close_hour=$5 WHERE id=$6", in.Name, in.Qualifications, in.Active, in.OpenHour, in.CloseHour, in.ID)
	}
	if e == nil {
		e = audit(ctx, tx, current(r).ID, nil, nil, "operator_updated", in.Name)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		fail(w, 500, "server_error", "Unable to save operator.")
		return
	}
	respond(w, 200, map[string]int64{"id": in.ID})
}
