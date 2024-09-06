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
	ADD IF NOT EXISTS open_host boolean DEFAULT false


CREATE TABLE IF NOT EXISTS public.mario_kart_wii_sake (
	regionid smallint NOT NULL CHECK (regionid >= 1 AND regionid <= 7),
	courseid smallint NOT NULL CHECK (courseid >= 0 AND courseid <= 32767),
	score integer NOT NULL CHECK (score > 0),
	pid integer NOT NULL CHECK (pid > 0),
	playerinfo varchar(108) NOT NULL CHECK (LENGTH(playerinfo) = 108),
	ghost bytea CHECK (ghost IS NULL OR (OCTET_LENGTH(ghost) BETWEEN 148 AND 10240)),
	
	CONSTRAINT one_time_per_course_constraint UNIQUE (courseid, pid)
);
	
	
ALTER TABLE public.mario_kart_wii_sake OWNER TO wiilink;
	
`)
}
