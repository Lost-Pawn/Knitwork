package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const userColumns = "id,name,email,role,verified,suspended,business,locale,qualifications,verification_note,allocation_group"

func (a *App) userByID(ctx context.Context, id int64) (*User, error) {
	u := &User{}
	e := a.db.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE id=$1", id).Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.Verified, &u.Suspended, &u.Business, &u.Locale, &u.Qualifications, &u.VerificationNote, &u.AllocationGroup)
	return u, e
}
func hashToken(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// The browser holds only a random session token; a database leak does not reveal usable cookies.
func (a *App) session(w http.ResponseWriter, r *http.Request, id int64) error {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return e
	}
	token := hex.EncodeToString(b)
	_, e := a.db.Exec(r.Context(), "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", hashToken(token), id, time.Now().Add(24*time.Hour))
	if e == nil {
		http.SetCookie(w, &http.Cookie{Name: "knitwork_session", Value: token, Path: "/", HttpOnly: true, Secure: a.production, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	}
	return e
}
func (a *App) require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, e := r.Cookie("knitwork_session")
		if e != nil {
			fail(w, 401, "login_required", "Please sign in.")
			return
		}
		var id int64
		e = a.db.QueryRow(r.Context(), "SELECT user_id FROM sessions WHERE token_hash=$1 AND expires_at>now()", hashToken(c.Value)).Scan(&id)
		if e != nil {
			fail(w, 401, "login_required", "Please sign in again.")
			return
		}
		u, e := a.userByID(r.Context(), id)
		if e != nil || u.Suspended {
			fail(w, 403, "account_unavailable", "This account is unavailable.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	}
}

type attempt struct {
	n     int
	until time.Time
}

var authAttempts = struct {
	sync.Mutex
	m map[string]attempt
}{m: map[string]attempt{}}

func allowAttempt(key string) bool {
	authAttempts.Lock()
	defer authAttempts.Unlock()
	now := time.Now()
	if len(authAttempts.m) > 5000 {
		for k, v := range authAttempts.m {
			if now.After(v.until) {
				delete(authAttempts.m, k)
			}
		}
	}
	v := authAttempts.m[key]
	if now.After(v.until) {
		v = attempt{until: now.Add(5 * time.Minute)}
	}
	v.n++
	authAttempts.m[key] = v
	return v.n <= 15
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !allowAttempt(in.Email) {
		fail(w, 429, "try_later", "Too many attempts. Try again in five minutes.")
		return
	}
	var id int64
	var hash string
	var suspended bool
	e := a.db.QueryRow(r.Context(), "SELECT id,password_hash,suspended FROM users WHERE email=$1", in.Email).Scan(&id, &hash, &suspended)
	if e != nil || suspended || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(w, 401, "invalid_login", "Email or password is incorrect.")
		return
	}
	if a.session(w, r, id) != nil {
		fail(w, 500, "server_error", "Unable to sign in.")
		return
	}
	u, _ := a.userByID(r.Context(), id)
	respond(w, 200, u)
}
func (a *App) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Business string `json:"business"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	_, e := mail.ParseAddress(in.Email)
	if e != nil || !strings.Contains(in.Email, "@") || len(strings.TrimSpace(in.Name)) < 2 || len(in.Name) > 100 || len(in.Password) < 10 || len(in.Password) > 72 || len(in.Business) > 150 {
		fail(w, 400, "invalid_registration", "Use a valid email, name and a password of 10 to 72 characters.")
		return
	}
	if !allowAttempt("register:" + in.Email) {
		fail(w, 429, "try_later", "Try again later.")
		return
	}
	h, e := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if e != nil {
		fail(w, 500, "server_error", "Unable to register.")
		return
	}
	var id int64
	e = a.db.QueryRow(r.Context(), "INSERT INTO users(name,email,password_hash,business) VALUES($1,$2,$3,$4) RETURNING id", strings.TrimSpace(in.Name), in.Email, string(h), in.Business).Scan(&id)
	if e != nil {
		fail(w, 409, "email_unavailable", "This email cannot be registered.")
		return
	}
	if a.session(w, r, id) != nil {
		fail(w, 500, "server_error", "Account created. Please sign in.")
		return
	}
	u, _ := a.userByID(r.Context(), id)
	respond(w, 201, u)
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("knitwork_session"); e == nil {
		_, _ = a.db.Exec(r.Context(), "DELETE FROM sessions WHERE token_hash=$1", hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "knitwork_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.production, SameSite: http.SameSiteLaxMode})
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) me(w http.ResponseWriter, r *http.Request) { respond(w, 200, current(r)) }
func (a *App) profile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Business         string `json:"business"`
		Locale           string `json:"locale"`
		VerificationNote string `json:"verification_note"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Business) > 150 || len(in.VerificationNote) > 1000 || !strings.Contains("|en|hi|mr|bn|ta|te|", "|"+in.Locale+"|") {
		fail(w, 400, "invalid_input", "Check profile details.")
		return
	}
	_, e := a.db.Exec(r.Context(), "UPDATE users SET business=$1,locale=$2,verification_note=$3 WHERE id=$4", in.Business, in.Locale, in.VerificationNote, current(r).ID)
	if e != nil {
		fail(w, 500, "server_error", "Unable to save profile.")
		return
	}
	u, _ := a.userByID(r.Context(), current(r).ID)
	respond(w, 200, u)
}
