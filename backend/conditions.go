package main

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Photos are bounded and stored with the loan, so a container restart cannot lose handover evidence.
func saveCondition(ctx context.Context, tx pgx.Tx, actor, booking int64, phase string, in ActionInput) string {
	if !in.Checklist || len(strings.TrimSpace(in.Note)) < 3 {
		return "condition_required"
	}
	parts := strings.SplitN(in.Image, ",", 2)
	if len(parts) != 2 {
		return "photo_required"
	}
	data, e := base64.StdEncoding.DecodeString(parts[1])
	if e != nil || len(data) < 20 || len(data) > 2<<20 {
		return "invalid_photo"
	}
	mime := http.DetectContentType(data)
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		return "invalid_photo"
	}
	_, e = tx.Exec(ctx, "INSERT INTO conditions(booking_id,actor_id,phase,note,damaged,mime,image) VALUES($1,$2,$3,$4,$5,$6,$7)", booking, actor, phase, in.Note, in.Damaged, mime, data)
	if e != nil {
		return "condition_conflict"
	}
	return ""
}
func (a *App) conditionImage(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var mime string
	var borrower, owner int64
	e := a.db.QueryRow(r.Context(), `SELECT c.image,c.mime,b.borrower_id,res.owner_id FROM conditions c JOIN bookings b ON b.id=c.booking_id JOIN resources res ON res.id=b.resource_id WHERE c.id=$1`, idFrom(r)).Scan(&data, &mime, &borrower, &owner)
	if e != nil {
		fail(w, 404, "not_found", "Photo not found.")
		return
	}
	if current(r).ID != borrower && !owns(current(r), owner) {
		fail(w, 403, "forbidden", "This evidence is private.")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}
