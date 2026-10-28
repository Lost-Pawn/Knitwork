package main

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
	"strings"
)

func (a *App) seed(ctx context.Context) error {
	tx, e := a.db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(740282)"); e != nil {
		return e
	}
	for _, d := range helpCorpus {
		if _, e = tx.Exec(ctx, "INSERT INTO help_docs(title,body) VALUES($1,$2) ON CONFLICT(title) DO UPDATE SET body=excluded.body", d.Title, d.Body); e != nil {
			return e
		}
	}
	if email, password := os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD"); email != "" && password != "" {
		if len(password) < 12 || len(password) > 72 {
			return fmt.Errorf("ADMIN_PASSWORD must contain 12 to 72 characters")
		}
		var exists bool
		if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", strings.ToLower(email)).Scan(&exists); e != nil {
			return e
		}
		if !exists {
			h, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
			if _, e = tx.Exec(ctx, "INSERT INTO users(name,email,password_hash,role,verified) VALUES('Centre administrator',$1,$2,'admin',true)", strings.ToLower(email), string(h)); e != nil {
				return e
			}
		}
	}
	if os.Getenv("DEMO_SEED") == "true" && !a.production {
		h, e := bcrypt.GenerateFromPassword([]byte("KnitworkDemo2026!"), 10)
		if e != nil {
			return e
		}
		demo := []struct {
			name, email, role, business string
			verified                    bool
			qual                        []string
		}{{"Meera Desai", "manager@knitwork.test", "manager", "Sakhi Garment Centre", true, []string{}}, {"Asha Patil", "asha@knitwork.test", "borrower", "Asha Tailoring", true, []string{"sewing"}}, {"Ravi Shah", "ravi@knitwork.test", "borrower", "Ravi Garments", true, []string{"sewing", "embroidery"}}, {"Centre Administrator", "admin@knitwork.test", "admin", "Sakhi Garment Centre", true, []string{}}, {"Equipment Owner", "owner@knitwork.test", "owner", "Sakhi Equipment Trust", true, []string{}}}
		for _, d := range demo {
			if _, e = tx.Exec(ctx, "INSERT INTO users(name,email,password_hash,role,business,verified,qualifications) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(email) DO NOTHING", d.name, d.email, string(h), d.role, d.business, d.verified, d.qual); e != nil {
				return e
			}
		}
		var n int
		_ = tx.QueryRow(ctx, "SELECT count(*) FROM resources").Scan(&n)
		if n == 0 {
			var owner int64
			if e = tx.QueryRow(ctx, "SELECT id FROM users WHERE email='manager@knitwork.test'").Scan(&owner); e != nil {
				return e
			}
			seed := []struct {
				name, cat, desc, training, policy string
				rate                              int
			}{{"Embroidery studio", "embroidery", "Computerised embroidery equipment for small garment batches. An authorised operator can help you complete your session.", "embroidery", "optional", 180}, {"Sewing workstation", "sewing", "A practical workstation for stitching and garment finishing. Bring your own fabric and thread.", "sewing", "optional", 60}, {"Cutting table", "cutting", "A spacious shared cutting table for fabric and patterns. Keep the surface clear at handover.", "", "none", 30}}
			for _, d := range seed {
				if _, e = tx.Exec(ctx, `INSERT INTO resources(owner_id,name,category,description,location,rules,training,operator_policy,hourly_rate,deposit,open_hour,close_hour) VALUES($1,$2,$3,$4,'Sakhi Garment Centre, Pune','Bring your own materials. Follow staff instructions. Record condition before and after use. Return the workspace clean.',$5,$6,$7,300,8,20)`, owner, d.name, d.cat, d.desc, d.training, d.policy, d.rate); e != nil {
					return e
				}
			}
		}
		_ = tx.QueryRow(ctx, "SELECT count(*) FROM operators").Scan(&n)
		if n == 0 {
			if _, e = tx.Exec(ctx, "INSERT INTO operators(name,qualifications,open_hour,close_hour) VALUES('Sunita More',ARRAY['embroidery','sewing'],8,20),('Farah Shaikh',ARRAY['sewing'],9,18)"); e != nil {
				return e
			}
		}
	}
	return tx.Commit(ctx)
}
