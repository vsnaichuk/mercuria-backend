--
-- PostgreSQL database dump
--

-- Dumped from database version 15.8 (Homebrew)
-- Dumped by pg_dump version 15.8 (Homebrew)

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

/* Main function to generate a uuidv7 value with millisecond precision */
/* See the UUID Version 7 specification at https://www.rfc-editor.org/rfc/rfc9562#name-uuid-version-7 */
CREATE FUNCTION uuidv7() RETURNS uuid
AS $$
  -- Replace the first 48 bits of a uuidv4 with the current
  -- number of milliseconds since 1970-01-01 UTC
  -- and set the "ver" field to 7 by setting additional bits
  select encode(
    set_bit(
      set_bit(
        overlay(uuid_send(gen_random_uuid()) placing
	  substring(int8send((extract(epoch from clock_timestamp())*1000)::bigint) from 3)
	  from 1 for 6),
	52, 1),
      53, 1), 'hex')::uuid;
$$ LANGUAGE sql volatile;

COMMENT ON FUNCTION uuidv7() IS
'Generate a uuid-v7 value with a 48-bit timestamp (millisecond precision) and 74 bits of randomness';


--
-- Name: get_or_create_user(text, text, text, text); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.get_or_create_user(p_oauth_id text, p_name text, p_avatar_url text, p_email text) RETURNS TABLE(id uuid, name text, avatar_url text)
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- Try to insert a new user, ignore the insert if the user already exists
    INSERT INTO users (id, oauth_id, name, avatar_url, email)
    VALUES (uuidv7(), p_oauth_id, p_name, p_avatar_url, p_email)
    ON CONFLICT (email) DO NOTHING;

    -- Retrieve the user
    RETURN QUERY
    SELECT users.id, users.name, users.avatar_url
    FROM users
    WHERE users.email = p_email;
END;
$$;


ALTER FUNCTION public.get_or_create_user(p_oauth_id text, p_name text, p_avatar_url text, p_email text) OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.events (
    id uuid NOT NULL,
    name text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    owner uuid NOT NULL,
    image_url text
);


ALTER TABLE public.events OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id uuid NOT NULL,
    oauth_id text,
    name text NOT NULL,
    avatar_url text,
    email text NOT NULL
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: events events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);


--
-- Name: users unique_users_email; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT unique_users_email UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: events events_owner_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_owner_fkey FOREIGN KEY (owner) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--
