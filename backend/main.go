package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

func (a *App) migrate(ctx context.Context) error {
	tx, e := a.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(740280)"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version text PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now())"); e != nil {
		return e
	}
	files, _ := migrations.ReadDir("migrations")
	for _, f := range files {
		var done bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", f.Name()).Scan(&done); e != nil {
			return e
		}
		if done {
			continue
		}
		sql, e := migrations.ReadFile("migrations/" + f.Name())
		if e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, string(sql)); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", f.Name()); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func (a *App) handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if a.db.Ping(ctx) != nil {
			fail(w, 503, "database_unavailable", "Database is not ready.")
			return
		}
		respond(w, 200, map[string]string{"status": "ok"})
	})
	m.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]any{"demo": !a.production && os.Getenv("DEMO_SEED") == "true", "translation": os.Getenv("HELP_API_KEY") != "", "timezone": "Asia/Kolkata"})
	})
	m.HandleFunc("POST /api/auth/login", a.login)
	m.HandleFunc("POST /api/auth/register", a.register)
	m.HandleFunc("POST /api/auth/logout", a.logout)
	m.HandleFunc("GET /api/me", a.require(a.me))
	m.HandleFunc("PUT /api/me", a.require(a.profile))
	m.HandleFunc("GET /api/resources", a.require(a.resources))
	m.HandleFunc("POST /api/resources", a.require(a.createResource))
	m.HandleFunc("GET /api/resources/{id}/availability", a.require(a.availability))
	m.HandleFunc("PATCH /api/resources/{id}", a.require(a.maintenance))
	m.HandleFunc("GET /api/operators", a.require(a.operators))
	m.HandleFunc("POST /api/operators", a.require(a.saveOperator))
	m.HandleFunc("GET /api/bookings", a.require(a.bookings))
	m.HandleFunc("POST /api/bookings", a.require(a.createBooking))
	m.HandleFunc("GET /api/bookings/{id}", a.require(a.bookingDetail))
	m.HandleFunc("POST /api/bookings/{id}/actions", a.require(a.bookingAction))
	m.HandleFunc("GET /api/conditions/{id}/image", a.require(a.conditionImage))
	m.HandleFunc("GET /api/members", a.require(a.members))
	m.HandleFunc("PATCH /api/members/{id}", a.require(a.verifyMember))
	m.HandleFunc("POST /api/incidents", a.require(a.createIncident))
	m.HandleFunc("PATCH /api/incidents/{id}", a.require(a.resolveIncident))
	m.HandleFunc("GET /api/notifications", a.require(a.notifications))
	m.HandleFunc("POST /api/notifications/read", a.require(a.readNotifications))
	m.HandleFunc("GET /api/dashboard", a.require(a.dashboard))
	m.HandleFunc("GET /api/help/docs", a.require(a.helpDocs))
	m.HandleFunc("POST /api/help/chat", a.require(a.helpChat))
	m.HandleFunc("POST /api/translate", a.require(a.translate))
	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "not_found", "Endpoint not found.") })
	dir := os.Getenv("STATIC_DIR")
	if dir == "" {
		dir = "../frontend/out"
	}
	fs := http.FileServer(http.Dir(dir))
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		path := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if _, e := os.Stat(path); e != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				fail(w, 415, "json_required", "Use JSON for this request.")
				return
			}
			origin := r.Header.Get("Origin")
			if origin != "" {
				expected := a.origin
				if expected == "" {
					p, _ := url.Parse(origin)
					if p == nil || p.Host != r.Host {
						fail(w, 403, "origin_rejected", "Unrecognised request origin.")
						return
					}
				} else if origin != expected {
					fail(w, 403, "origin_rejected", "Unrecognised request origin.")
					return
				}
			}
		}
		m.ServeHTTP(w, r)
	})
}
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, e := pgxpool.New(ctx, dsn)
	if e != nil {
		log.Fatal("Invalid database configuration")
	}
	defer db.Close()
	a := &App{db: db, origin: strings.TrimRight(os.Getenv("APP_ORIGIN"), "/"), production: os.Getenv("APP_ENV") == "production"}
	if a.origin == "" && os.Getenv("RENDER_EXTERNAL_URL") != "" {
		a.origin = os.Getenv("RENDER_EXTERNAL_URL")
	}
	if a.production && (a.origin == "" || !strings.HasPrefix(a.origin, "https://") || os.Getenv("DEMO_SEED") == "true") {
		log.Fatal("Production requires an HTTPS APP_ORIGIN or RENDER_EXTERNAL_URL and DEMO_SEED=false")
	}
	if e = a.migrate(ctx); e != nil {
		log.Fatalf("Migration failed: %v", e)
	}
	if e = a.seed(ctx); e != nil {
		log.Fatalf("Initialisation failed: %v", e)
	}
	go a.reminders(ctx)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := http.Server{Addr: ":" + port, Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		s, c := context.WithTimeout(context.Background(), 8*time.Second)
		defer c()
		_ = srv.Shutdown(s)
	}()
	log.Printf("Knitwork listening on %s", fmt.Sprintf("http://localhost:%s", port))
	if e = srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}
