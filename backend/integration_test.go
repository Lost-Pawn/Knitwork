package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFullSharingLifecycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for integration tests")
	}
	if !strings.Contains(strings.Split(dsn, "?")[0], "/knitwork_test") {
		t.Fatal("refusing to reset a non-test database")
	}
	ctx := context.Background()
	db, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	a := &App{db: db}
	if e = a.migrate(ctx); e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec(ctx, "TRUNCATE notifications,audit,incidents,conditions,bookings,operators,resources,sessions,users,help_docs RESTART IDENTITY CASCADE")
	if e != nil {
		t.Fatal(e)
	}
	t.Setenv("DEMO_SEED", "true")
	t.Setenv("ADMIN_EMAIL", "")
	t.Setenv("ADMIN_PASSWORD", "")
	if e = a.seed(ctx); e != nil {
		t.Fatal(e)
	}
	handler := a.handler()
	call := func(cookie *http.Cookie, method, path string, payload any) (int, []byte, *http.Cookie) {
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var c *http.Cookie
		if cs := w.Result().Cookies(); len(cs) > 0 {
			c = cs[0]
		}
		return w.Code, w.Body.Bytes(), c
	}
	must := func(cookie *http.Cookie, method, path string, payload any, want int) []byte {
		t.Helper()
		code, body, _ := call(cookie, method, path, payload)
		if code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, code, want, body)
		}
		return body
	}
	login := func(email string) *http.Cookie {
		t.Helper()
		code, body, c := call(nil, "POST", "/api/auth/login", map[string]string{"email": email, "password": "KnitworkDemo2026!"})
		if code != 200 {
			t.Fatalf("login: %d %s", code, body)
		}
		return c
	}
	manager := login("manager@knitwork.test")
	asha := login("asha@knitwork.test")
	ravi := login("ravi@knitwork.test")
	var managerUser, ashaUser, raviUser User
	json.Unmarshal(must(manager, "GET", "/api/me", nil, 200), &managerUser)
	json.Unmarshal(must(asha, "GET", "/api/me", nil, 200), &ashaUser)
	json.Unmarshal(must(ravi, "GET", "/api/me", nil, 200), &raviUser)
	start := time.Now().In(india).AddDate(0, 0, 1)
	start = time.Date(start.Year(), start.Month(), start.Day(), 10, 0, 0, 0, india)
	request := func(cookie *http.Cookie, resource int64, s time.Time, assisted bool) int64 {
		t.Helper()
		body := must(cookie, "POST", "/api/bookings", BookingInput{ResourceID: resource, StartsAt: s, EndsAt: s.Add(time.Hour), Purpose: "Prepare a small garment batch", Assisted: assisted, Accepted: true}, 201)
		var out map[string]int64
		json.Unmarshal(body, &out)
		return out["id"]
	}
	act := func(cookie *http.Cookie, id int64, in ActionInput, want int) []byte {
		t.Helper()
		return must(cookie, "POST", fmt.Sprintf("/api/bookings/%d/actions", id), in, want)
	}
	var imageBuf bytes.Buffer
	_ = png.Encode(&imageBuf, image.NewRGBA(image.Rect(0, 0, 12, 12)))
	photo := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageBuf.Bytes())
	operator := int64(1)
	t.Run("authentication and role boundaries", func(t *testing.T) {
		must(nil, "GET", "/api/resources", nil, 401)
		must(asha, "GET", "/api/members", nil, 403)
		must(asha, "POST", "/api/resources", ResourceInput{}, 403)
		must(nil, "POST", "/api/auth/register", map[string]any{"name": "Mallory", "email": "mallory@example.test", "password": "VeryLongPassword", "role": "admin"}, 400)
		r := httptest.NewRequest("POST", "/api/auth/logout", strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "https://untrusted.example")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("cross origin request accepted")
		}
	})
	var first int64
	t.Run("supervision approval private handover and return", func(t *testing.T) {
		first = request(asha, 1, start, true)
		act(manager, first, ActionInput{Action: "approve", Reason: "Membership checked"}, 409)
		act(manager, first, ActionInput{Action: "approve", OperatorID: &operator, Reason: "Operator assigned"}, 200)
		must(ravi, "GET", fmt.Sprintf("/api/bookings/%d", first), nil, 403)
		act(asha, first, ActionInput{Action: "checkin", Note: "All parts look sound", Checklist: true, Image: photo}, 409)
		act(manager, first, ActionInput{Action: "checkin", Reason: "Early supervised handover recorded", Note: "All parts look sound", Checklist: true, Image: photo}, 200)
		act(asha, first, ActionInput{Action: "return", Reason: "Session finished", Note: "No damage, work surface clean", Checklist: true, Image: photo}, 200)
		var detail struct {
			Conditions []struct {
				ID int64 `json:"id"`
			}
		}
		json.Unmarshal(must(asha, "GET", fmt.Sprintf("/api/bookings/%d", first), nil, 200), &detail)
		if len(detail.Conditions) != 2 {
			t.Fatal("missing handover photos")
		}
		must(ravi, "GET", fmt.Sprintf("/api/conditions/%d/image", detail.Conditions[0].ID), nil, 403)
		must(asha, "GET", fmt.Sprintf("/api/conditions/%d/image", detail.Conditions[0].ID), nil, 200)
		act(manager, first, ActionInput{Action: "confirm_return", Reason: "Inspected and accepted clean return"}, 200)
	})
	t.Run("concurrent approvals cannot double book", func(t *testing.T) {
		b1 := request(asha, 2, start, true)
		b2 := request(ravi, 2, start, false)
		var codes [2]int
		var wg sync.WaitGroup
		for i, id := range []int64{b1, b2} {
			wg.Add(1)
			go func(i int, id int64) {
				defer wg.Done()
				codes[i], _, _ = call(manager, "POST", fmt.Sprintf("/api/bookings/%d/actions", id), ActionInput{Action: "approve", Reason: "Concurrent review", OperatorID: &operator})
			}(i, id)
		}
		wg.Wait()
		if !((codes[0] == 200 && codes[1] == 409) || (codes[0] == 409 && codes[1] == 200)) {
			t.Fatalf("unexpected concurrent outcomes %v", codes)
		}
		for i, id := range []int64{b1, b2} {
			if codes[i] == 200 {
				act(manager, id, ActionInput{Action: "cancel", Reason: "Free slot after concurrency test"}, 200)
			} else {
				act(manager, id, ActionInput{Action: "reject", Reason: "Competing request reviewed"}, 200)
			}
		}
	})
	t.Run("waiting list prevents unexplained queue jumping", func(t *testing.T) {
		b1 := request(asha, 3, start, false)
		act(manager, b1, ActionInput{Action: "waitlist", Reason: "Waiting for available session"}, 200)
		b2 := request(ravi, 3, start, false)
		act(manager, b2, ActionInput{Action: "approve", Reason: "Ordinary review"}, 409)
		act(manager, b2, ActionInput{Action: "approve", Reason: "Exceptional deadline reviewed by manager", Override: true}, 200)
		act(manager, b2, ActionInput{Action: "cancel", Reason: "Release reservation for waiting member"}, 200)
		act(manager, b1, ActionInput{Action: "approve", Reason: "First eligible waiting member"}, 200)
		act(manager, b1, ActionInput{Action: "cancel", Reason: "End queue test"}, 200)
	})
	t.Run("extension conflicts and damaged return", func(t *testing.T) {
		b1 := request(asha, 1, start.Add(3*time.Hour), true)
		b2 := request(ravi, 1, start.Add(5*time.Hour), false)
		act(manager, b1, ActionInput{Action: "approve", OperatorID: &operator, Reason: "Supervised session approved"}, 200)
		act(manager, b2, ActionInput{Action: "approve", Reason: "Training verified"}, 200)
		act(asha, b1, ActionInput{Action: "extend", End: start.Add(6 * time.Hour), Reason: "Need extra work time"}, 200)
		act(manager, b1, ActionInput{Action: "approve_extension", Reason: "Check competing session"}, 409)
		act(manager, b1, ActionInput{Action: "reject_extension", Reason: "Another member has reserved this time"}, 200)
		act(manager, b1, ActionInput{Action: "checkin", Reason: "Early demonstration handover", Note: "Initial condition documented", Image: photo, Checklist: true}, 200)
		act(asha, b1, ActionInput{Action: "return", Reason: "Equipment returned with fault", Note: "Loose fixture requires inspection", Image: photo, Checklist: true, Damaged: true}, 200)
		must(manager, "PATCH", "/api/resources/1", map[string]any{"maintenance": false, "reason": "Attempt early release"}, 409)
		act(manager, b1, ActionInput{Action: "confirm_return", Reason: "Return accepted; equipment remains held"}, 200)
		var d struct {
			Incidents []struct {
				ID int64 `json:"id"`
			}
		}
		json.Unmarshal(must(manager, "GET", fmt.Sprintf("/api/bookings/%d", b1), nil, 200), &d)
		must(manager, "PATCH", fmt.Sprintf("/api/incidents/%d", d.Incidents[0].ID), map[string]string{"status": "resolved", "resolution": "Fixture repaired and inspected"}, 200)
		must(manager, "PATCH", "/api/resources/1", map[string]any{"maintenance": false, "reason": "Post repair inspection passed"}, 200)
	})
	t.Run("unverified member cannot receive approval", func(t *testing.T) {
		body := must(nil, "POST", "/api/auth/register", map[string]string{"name": "New Member", "email": "new@example.test", "password": "NewMemberPassword", "business": "New Tailoring"}, 201)
		var u User
		json.Unmarshal(body, &u)
		b := must(manager, "POST", "/api/bookings", BookingInput{ResourceID: 3, BorrowerID: u.ID, StartsAt: start.Add(6 * time.Hour), EndsAt: start.Add(7 * time.Hour), Purpose: "Assisted request recorded", Accepted: true}, 201)
		var id map[string]int64
		json.Unmarshal(b, &id)
		act(manager, id["id"], ActionInput{Action: "approve", Reason: "Verification still pending"}, 409)
	})
	t.Run("help retrieval cites sources and unsupported query declines", func(t *testing.T) {
		var answer struct {
			Answer  string    `json:"answer"`
			Sources []HelpDoc `json:"sources"`
			Mode    string    `json:"mode"`
		}
		json.Unmarshal(must(asha, "POST", "/api/help/chat", map[string]string{"question": "How do I return damaged equipment?", "locale": "en"}, 200), &answer)
		if len(answer.Sources) == 0 || answer.Mode != "local_retrieval" {
			t.Fatal("retrieval missing")
		}
		json.Unmarshal(must(asha, "POST", "/api/help/chat", map[string]string{"question": "quantum quasiparticle neutrino", "locale": "en"}, 200), &answer)
		if len(answer.Sources) != 0 || !strings.Contains(answer.Answer, "manager") {
			t.Fatal("unsupported question was answered")
		}
	})
	t.Run("reminders are idempotent", func(t *testing.T) {
		var before, after int
		_ = db.QueryRow(ctx, "SELECT count(*) FROM notifications").Scan(&before)
		if e = a.reminderTick(ctx); e != nil {
			t.Fatal(e)
		}
		if e = a.reminderTick(ctx); e != nil {
			t.Fatal(e)
		}
		_ = db.QueryRow(ctx, "SELECT count(*) FROM notifications").Scan(&after)
		if after < before {
			t.Fatal("notification loss")
		}
	})
	t.Run("logout revokes session", func(t *testing.T) {
		must(asha, "POST", "/api/auth/logout", map[string]bool{}, 200)
		must(asha, "GET", "/api/me", nil, 401)
	})
}
