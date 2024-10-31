SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', 'public', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;
SET default_tablespace = '';
SET default_table_access_method = heap;

/* Main function to generate a uuidv7 value with millisecond precision */
/* See the UUID Version 7 specification at https://www.rfc-editor.org/rfc/rfc9562#name-uuid-version-7 */
CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid
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

CREATE FUNCTION get_or_create_user(p_oauth_id text, p_name text, p_avatar_url text, p_email text) RETURNS TABLE(id uuid, name text, avatar_url text)
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

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE users (
    id uuid PRIMARY KEY,
    oauth_id text,
    name text NOT NULL,
    avatar_url text,
    email text NOT NULL,
    CONSTRAINT unique_users_email UNIQUE (email)
);


--
-- Name: events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE events (
    id uuid PRIMARY KEY,
    name text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    owner uuid NOT NULL,
    image_url text,
    CONSTRAINT events_owner_fkey FOREIGN KEY (owner) REFERENCES users(id)
);

--
-- Name: likes; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE likes (
    id BIGSERIAL PRIMARY KEY,
    user_id uuid NOT NULL,
    event_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT likes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT likes_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);

--
-- Name: members; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE members (
    id BIGSERIAL PRIMARY KEY,
    user_id uuid NOT NULL,
    event_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT "now"() NOT NULL,
    CONSTRAINT members_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT members_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON UPDATE CASCADE ON DELETE CASCADE
);

--
-- Name: invites; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE invites (
    id uuid DEFAULT uuidv7() PRIMARY KEY,
    event_id uuid NOT NULL,
    status VARCHAR(50) DEFAULT 'Pending' NOT NULL CHECK (status IN ('Pending', 'Accepted', 'Declined')),
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone DEFAULT (now() + INTERVAL '24 hours') NOT NULL,
    CONSTRAINT invites_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT invites_event_id_fkey FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);

-- TYPE: event_type

CREATE TYPE event_type AS (
    id uuid,
    name text,
    created_at timestamp with time zone,
    owner uuid,
    image_url text,
    owner_id uuid,
    owner_oauth_id text,
    owner_name text,
    owner_avatar_url text,
    owner_email text,
    like_id bigint,
    like_user_id uuid,
    like_event_id uuid,
    like_created_at timestamp with time zone,
    member_id uuid,
    member_oauth_id text,
    member_name text,
    member_avatar_url text,
    member_email text
);
-- FUNCTION: get_event(uuid)

CREATE OR REPLACE FUNCTION get_event(
    _id uuid)
    RETURNS SETOF event_type
    LANGUAGE 'plpgsql'

AS $BODY$
BEGIN
    RETURN QUERY
    SELECT events.*, users.*, likes.*, users_mbr.*
    FROM events

    INNER JOIN users ON users.id = events.owner
    LEFT JOIN likes ON likes.event_id = events.id
    INNER JOIN members ON members.event_id = events.id
    INNER JOIN users AS users_mbr ON users_mbr.id = members.user_id
    WHERE events.id = _id;
END;
$BODY$;

-- FUNCTION: create_event(text, uuid)

CREATE OR REPLACE FUNCTION create_event(
    _name text,
    _owner uuid)
    RETURNS uuid
    LANGUAGE 'plpgsql'

AS $BODY$
DECLARE
    new_event_id uuid;
BEGIN
    INSERT INTO events AS e (id, name, created_at, owner, image_url)
    VALUES (uuidv7(), _name, now(), _owner, '')
    RETURNING e.id INTO new_event_id;

    -- add owner to members
    INSERT INTO members (user_id, event_id)
    VALUES (_owner, new_event_id);

    RETURN new_event_id;
END;
$BODY$;
-- FUNCTION: get_events(text, uuid)

CREATE OR REPLACE FUNCTION get_events(_user_id uuid)
  RETURNS SETOF event_type
  LANGUAGE 'plpgsql'

AS $BODY$
BEGIN
    RETURN QUERY
    WITH user_events AS (
        SELECT events.*
        FROM events
        JOIN members ON members.event_id = events.id
        WHERE members.user_id = _user_id
    )
    SELECT user_events.*, users.*, likes.*, users_mbr.*
    FROM user_events

    INNER JOIN users ON users.id = user_events.owner
    LEFT JOIN likes ON likes.event_id = user_events.id
    INNER JOIN members ON members.event_id = user_events.id
    INNER JOIN users AS users_mbr ON users_mbr.id = members.user_id;
END;
$BODY$;
