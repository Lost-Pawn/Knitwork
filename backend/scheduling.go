package main

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

var india = time.FixedZone("IST", 19800)

type resourcePolicy struct {
	ID, Owner                              int64
	Training, OperatorPolicy               string
	Maintenance                            bool
	Max, Weekly, Open, Close, Buffer, Rate int
	From, To                               time.Time
}
type bookingRecord struct {
	ID, Resource, Borrower int64
	Operator               *int64
	Start, End             time.Time
	Status                 string
	Assisted               bool
	Extension              *time.Time
}

func loadPolicy(ctx context.Context, tx pgx.Tx, id int64) (p resourcePolicy, e error) {
	p.ID = id
	e = tx.QueryRow(ctx, `SELECT owner_id,training,operator_policy,maintenance,max_hours,weekly_hours,open_hour,close_hour,buffer_minutes,hourly_rate,available_from,available_to FROM resources WHERE id=$1 FOR UPDATE`, id).Scan(&p.Owner, &p.Training, &p.OperatorPolicy, &p.Maintenance, &p.Max, &p.Weekly, &p.Open, &p.Close, &p.Buffer, &p.Rate, &p.From, &p.To)
	return
}
func loadBooking(ctx context.Context, tx pgx.Tx, id int64) (b bookingRecord, e error) {
	b.ID = id
	e = tx.QueryRow(ctx, `SELECT resource_id,borrower_id,operator_id,starts_at,ends_at,status,assisted,extension_end FROM bookings WHERE id=$1 FOR UPDATE`, id).Scan(&b.Resource, &b.Borrower, &b.Operator, &b.Start, &b.End, &b.Status, &b.Assisted, &b.Extension)
	return
}
func withinHours(start, end time.Time, open, close int) bool {
	s, e := start.In(india), end.In(india)
	return s.Format("2006-01-02") == e.Format("2006-01-02") && s.Hour() >= open && float64(e.Hour())+float64(e.Minute())/60 <= float64(close)
}
func checkTime(p resourcePolicy, start, end time.Time) error {
	if !end.After(start) || end.Sub(start) < 15*time.Minute || end.Sub(start) > time.Duration(p.Max)*time.Hour {
		return fmt.Errorf("duration_limit")
	}
	if !withinHours(start, end, p.Open, p.Close) {
		return fmt.Errorf("outside_hours")
	}
	day := start.In(india).Format("2006-01-02")
	if day < p.From.Format("2006-01-02") || day > p.To.Format("2006-01-02") {
		return fmt.Errorf("outside_dates")
	}
	return nil
}
func weekStart(t time.Time) time.Time {
	t = t.In(india)
	d := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-d, 0, 0, 0, 0, india)
}

// This lock serialises allocation decisions across resources, including a shared operator and member quota.
// Database exclusion constraints remain the final defence when multiple app instances are running.
func allocationLock(ctx context.Context, tx pgx.Tx) error {
	_, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(740281)")
	return e
}
func checkAllocation(ctx context.Context, tx pgx.Tx, p resourcePolicy, b bookingRecord, operator *int64, override bool, reason string) error {
	if p.Maintenance {
		return fmt.Errorf("maintenance")
	}
	if e := checkTime(p, b.Start, b.End); e != nil {
		return e
	}
	var verified, suspended bool
	var quals []string
	if e := tx.QueryRow(ctx, "SELECT verified,suspended,qualifications FROM users WHERE id=$1 FOR UPDATE", b.Borrower).Scan(&verified, &suspended, &quals); e != nil {
		return e
	}
	if !verified || suspended {
		return fmt.Errorf("verification_required")
	}
	needsOperator := p.OperatorPolicy == "required" || b.Assisted
	if p.Training != "" && !slices.Contains(quals, p.Training) {
		if p.OperatorPolicy == "none" || !b.Assisted {
			return fmt.Errorf("training_required")
		}
		needsOperator = true
	}
	if needsOperator && operator == nil {
		return fmt.Errorf("operator_required")
	}
	if operator != nil {
		var active bool
		var oq []string
		var open, close int
		if e := tx.QueryRow(ctx, "SELECT active,qualifications,open_hour,close_hour FROM operators WHERE id=$1 FOR UPDATE", *operator).Scan(&active, &oq, &open, &close); e != nil {
			return fmt.Errorf("operator_unavailable")
		}
		if !active || !withinHours(b.Start, b.End, open, close) || (p.Training != "" && !slices.Contains(oq, p.Training)) {
			return fmt.Errorf("operator_unavailable")
		}
	}
	var used float64
	week := weekStart(b.Start)
	if e := tx.QueryRow(ctx, `SELECT COALESCE(sum(extract(epoch from(ends_at-starts_at))/3600),0) FROM bookings WHERE (borrower_id=$1 OR borrower_id IN (SELECT id FROM users WHERE allocation_group<>'' AND allocation_group=(SELECT allocation_group FROM users WHERE id=$1))) AND resource_id=$2 AND id<>$3 AND starts_at>=$4 AND starts_at<$5 AND status IN ('approved','in_use','return_pending','completed')`, b.Borrower, b.Resource, b.ID, week, week.AddDate(0, 0, 7)).Scan(&used); e != nil {
		return e
	}
	if used+b.End.Sub(b.Start).Hours() > float64(p.Weekly) {
		return fmt.Errorf("weekly_limit")
	}
	busy := b.End.Add(time.Duration(p.Buffer) * time.Minute)
	var n int
	if e := tx.QueryRow(ctx, `SELECT count(*) FROM bookings WHERE id<>$1 AND status IN ('approved','in_use','return_pending') AND starts_at<$3 AND busy_until>$2 AND (resource_id=$4 OR ($5::bigint IS NOT NULL AND operator_id=$5))`, b.ID, b.Start, busy, b.Resource, operator).Scan(&n); e != nil {
		return e
	}
	if n > 0 {
		return fmt.Errorf("schedule_conflict")
	}
	// Only an eligible earlier request blocks queue order. A manager can document an exceptional allocation.
	var earlier int
	if e := tx.QueryRow(ctx, `SELECT count(*) FROM bookings q JOIN users u ON u.id=q.borrower_id WHERE q.resource_id=$1 AND q.id<>$2 AND q.status='waitlisted' AND q.starts_at<$4 AND q.busy_until>$3 AND (q.created_at,q.id)<(SELECT created_at,id FROM bookings WHERE id=$2) AND u.verified AND NOT u.suspended AND ($5='' OR $5=ANY(u.qualifications) OR (q.assisted AND $6<>'none'))`, b.Resource, b.ID, b.Start, busy, p.Training, p.OperatorPolicy).Scan(&earlier); e != nil {
		return e
	}
	if earlier > 0 && (!override || len(reason) < 5) {
		return fmt.Errorf("queue_priority")
	}
	return nil
}
func estimate(rate int, start, end time.Time) int {
	return int(math.Ceil(end.Sub(start).Hours() * float64(rate)))
}

var allocationMessages = map[string]string{"maintenance": "Equipment is awaiting maintenance review.", "duration_limit": "Choose a session within the equipment duration limit.", "outside_hours": "Choose a time within the centre operating hours.", "outside_dates": "Choose a date inside the listed availability window.", "verification_required": "The borrower must be verified and active.", "training_required": "Required training has not been verified. Request permitted supervised access or contact the manager.", "operator_required": "Assign an authorised operator before approval.", "operator_unavailable": "The selected operator is unavailable or lacks the required authorisation.", "weekly_limit": "This booking would exceed the member's weekly hours limit.", "schedule_conflict": "Equipment or operator is already reserved during this period.", "queue_priority": "An earlier eligible request is waiting for this slot. Review it first or record an exception reason."}
