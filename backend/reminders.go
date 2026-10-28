package main

import (
	"context"
	"log"
	"time"
)

func (a *App) reminderTick(ctx context.Context) error {
	_, e := a.db.Exec(ctx, `INSERT INTO notifications(user_id,booking_id,kind,message,dedupe_key) SELECT borrower_id,id,'reminder','Your session starts within an hour. Review collection instructions.','reminder:'||id||':'||starts_at FROM bookings WHERE status='approved' AND starts_at BETWEEN now() AND now()+interval '1 hour' ON CONFLICT(dedupe_key) DO NOTHING`)
	if e != nil {
		return e
	}
	_, e = a.db.Exec(ctx, `INSERT INTO notifications(user_id,booking_id,kind,message,dedupe_key) SELECT borrower_id,id,'overdue','Your equipment session is overdue. Contact the manager and arrange return.','overdue:'||id||':'||ends_at FROM bookings WHERE status='in_use' AND ends_at<now() ON CONFLICT(dedupe_key) DO NOTHING`)
	if e != nil {
		return e
	}
	_, e = a.db.Exec(ctx, `INSERT INTO notifications(user_id,booking_id,kind,message,dedupe_key) SELECT r.owner_id,b.id,'overdue','A borrower is overdue. Review the handover.','owner-overdue:'||b.id||':'||b.ends_at FROM bookings b JOIN resources r ON r.id=b.resource_id WHERE b.status='in_use' AND b.ends_at<now() ON CONFLICT(dedupe_key) DO NOTHING`)
	if e == nil {
		_, e = a.db.Exec(ctx, "DELETE FROM sessions WHERE expires_at<now()")
	}
	return e
}
func (a *App) reminders(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if e := a.reminderTick(ctx); e != nil {
			log.Printf("Reminder worker: %v", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
