package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	db         *pgxpool.Pool
	origin     string
	production bool
}
type User struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	Role             string   `json:"role"`
	Verified         bool     `json:"verified"`
	Suspended        bool     `json:"suspended"`
	Business         string   `json:"business"`
	Locale           string   `json:"locale"`
	Qualifications   []string `json:"qualifications"`
	AllocationGroup  string   `json:"allocation_group"`
	VerificationNote string   `json:"verification_note"`
}
type ResourceInput struct {
	Name           string `json:"name"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Location       string `json:"location"`
	Rules          string `json:"rules"`
	Training       string `json:"training"`
	OperatorPolicy string `json:"operator_policy"`
	HourlyRate     int    `json:"hourly_rate"`
	Deposit        int    `json:"deposit"`
	MaxHours       int    `json:"max_hours"`
	WeeklyHours    int    `json:"weekly_hours"`
	OpenHour       int    `json:"open_hour"`
	CloseHour      int    `json:"close_hour"`
	BufferMinutes  int    `json:"buffer_minutes"`
	AvailableFrom  string `json:"available_from"`
	AvailableTo    string `json:"available_to"`
}
type BookingInput struct {
	ResourceID int64     `json:"resource_id"`
	BorrowerID int64     `json:"borrower_id"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	Purpose    string    `json:"purpose"`
	Assisted   bool      `json:"assisted"`
	Accepted   bool      `json:"accepted"`
}
type ActionInput struct {
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	OperatorID *int64    `json:"operator_id"`
	End        time.Time `json:"end"`
	Note       string    `json:"note"`
	Image      string    `json:"image"`
	Damaged    bool      `json:"damaged"`
	Checklist  bool      `json:"checklist"`
	Override   bool      `json:"override"`
}
type apiError struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}
type userKey struct{}

func current(r *http.Request) *User  { u, _ := r.Context().Value(userKey{}).(*User); return u }
func staff(u *User) bool             { return u != nil && (u.Role == "manager" || u.Role == "admin") }
func owns(u *User, owner int64) bool { return staff(u) || (u.Role == "owner" && u.ID == owner) }
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, code, msg string) {
	respond(w, status, apiError{code, msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		fail(w, 400, "invalid_input", "Check the submitted fields.")
		return false
	}
	return true
}
func listJSON(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, sql string, args ...any) ([]json.RawMessage, error) {
	rows, e := q.Query(ctx, sql, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var v json.RawMessage
		if e = rows.Scan(&v); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func audit(ctx context.Context, tx pgx.Tx, actor int64, booking, resource *int64, action, detail string) error {
	_, e := tx.Exec(ctx, "INSERT INTO audit(actor_id,booking_id,resource_id,action,detail) VALUES($1,$2,$3,$4,$5)", actor, booking, resource, action, detail)
	return e
}
func notify(ctx context.Context, tx pgx.Tx, user, booking int64, kind, message string) error {
	_, e := tx.Exec(ctx, "INSERT INTO notifications(user_id,booking_id,kind,message) VALUES($1,$2,$3,$4)", user, booking, kind, message)
	return e
}
