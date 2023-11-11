--
-- PostgreSQL database dump
--

-- Dumped from database version 14.9 (Ubuntu 14.9-0ubuntu0.22.04.1)
-- Dumped by pg_dump version 14.9 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: logins; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE public.logins (
    id integer NOT NULL,
    auth_token character varying NOT NULL,
    user_id bigint NOT NULL,
    gsbrcd character varying,
    challenge character varying
);


ALTER TABLE public.logins OWNER TO wiilink;

--
-- Name: logins_id_seq; Type: SEQUENCE; Schema: public; Owner: wiilink
--

CREATE SEQUENCE public.logins_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.logins_id_seq OWNER TO wiilink;

--
-- Name: logins_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: wiilink
--

ALTER SEQUENCE public.logins_id_seq OWNED BY public.logins.id;


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE public.sessions (
    id integer NOT NULL,
    session_key character varying NOT NULL,
    profile_id integer NOT NULL,
    login_ticket character varying NOT NULL
);


ALTER TABLE public.sessions OWNER TO wiilink;

--
-- Name: sessions_id_seq; Type: SEQUENCE; Schema: public; Owner: wiilink
--

CREATE SEQUENCE public.sessions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.sessions_id_seq OWNER TO wiilink;

--
-- Name: sessions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: wiilink
--

ALTER SEQUENCE public.sessions_id_seq OWNED BY public.sessions.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: wiilink
--

CREATE TABLE public.users (
    profile_id integer NOT NULL,
    user_id bigint NOT NULL,
    gsbrcd character varying NOT NULL,
    password character varying NOT NULL,
    email character varying NOT NULL,
    unique_nick character varying NOT NULL,
    firstname character varying,
    lastname character varying DEFAULT ''::character varying,
    mariokartwii_friend_info character varying
);


ALTER TABLE public.users OWNER TO wiilink;

--
-- Name: users_profile_id_seq; Type: SEQUENCE; Schema: public; Owner: wiilink
--

CREATE SEQUENCE public.users_profile_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.users_profile_id_seq OWNER TO wiilink;

--
-- Name: users_profile_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: wiilink
--

ALTER SEQUENCE public.users_profile_id_seq OWNED BY public.users.profile_id;


--
-- Name: logins id; Type: DEFAULT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.logins ALTER COLUMN id SET DEFAULT nextval('public.logins_id_seq'::regclass);


--
-- Name: sessions id; Type: DEFAULT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.sessions ALTER COLUMN id SET DEFAULT nextval('public.sessions_id_seq'::regclass);


--
-- Name: users profile_id; Type: DEFAULT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.users ALTER COLUMN profile_id SET DEFAULT nextval('public.users_profile_id_seq'::regclass);


--
-- Name: logins logins_auth_token_key; Type: CONSTRAINT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.logins
    ADD CONSTRAINT logins_auth_token_key UNIQUE (auth_token);


--
-- Name: logins logins_pkey; Type: CONSTRAINT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.logins
    ADD CONSTRAINT logins_pkey PRIMARY KEY (id);


--
-- Name: logins logins_user_id_key; Type: CONSTRAINT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.logins
    ADD CONSTRAINT logins_user_id_key UNIQUE (user_id, gsbrcd);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: wiilink
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (profile_id);


--
-- PostgreSQL database dump complete
--

