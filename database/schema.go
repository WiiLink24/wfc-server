package database

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

func UpdateTables(pool *pgxpool.Pool, ctx context.Context) {
	pool.Exec(ctx, `

ALTER TABLE ONLY public.users
	ADD IF NOT EXISTS last_ip_address character varying DEFAULT ''::character varying,
	ADD IF NOT EXISTS last_ingamesn character varying DEFAULT ''::character varying,
	ADD IF NOT EXISTS has_ban boolean DEFAULT false,
	ADD IF NOT EXISTS ban_issued timestamp without time zone,
	ADD IF NOT EXISTS ban_expires timestamp without time zone,
	ADD IF NOT EXISTS ban_reason character varying,
	ADD IF NOT EXISTS ban_reason_hidden character varying,
	ADD IF NOT EXISTS ban_moderator character varying,
	ADD IF NOT EXISTS ban_tos boolean,
	ADD IF NOT EXISTS open_host boolean DEFAULT false;

	`)

	pool.Exec(ctx, `

	ALTER TABLE ONLY public.mario_kart_wii_sake
		ADD IF NOT EXISTS upload_time timestamp without time zone;
	
	`)
}
