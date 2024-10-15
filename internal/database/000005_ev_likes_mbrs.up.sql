CREATE TABLE likes (
    id BIGSERIAL PRIMARY KEY,
    user_id uuid NOT NULL,
    event_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT "now"() NOT NULL
);
ALTER TABLE ONLY public.likes
    ADD CONSTRAINT likes_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.likes
    ADD CONSTRAINT likes_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.events(id) ON DELETE CASCADE;


CREATE TABLE members (
    id BIGSERIAL PRIMARY KEY,
    user_id uuid NOT NULL,
    event_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT "now"() NOT NULL
);
ALTER TABLE ONLY public.members
    ADD CONSTRAINT members_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON UPDATE CASCADE ON DELETE CASCADE;
ALTER TABLE ONLY public.members
    ADD CONSTRAINT members_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.events(id) ON UPDATE CASCADE ON DELETE CASCADE;

-- DROP TYPE IF EXISTS event_type CASCADE;
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
    -- UPD WITH LIKES
    like_id bigint,
    like_user_id uuid,
    like_event_id uuid,
    like_created_at timestamp with time zone,
    -- UPD WITH MEMBERS
    member_id uuid,
    member_oauth_id text,
    member_name text,
    member_avatar_url text,
    member_email text
);

-- UPD(OWNER DATA, LIKES, MEMBERS)
-- DROP FUNCTION IF EXISTS get_events CASCADE;
CREATE OR REPLACE FUNCTION public.get_events(_user_id uuid)
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

-- UPD(OWNER DATA, LIKES, MEMBERS)
-- DROP FUNCTION IF EXISTS get_event CASCADE;
CREATE OR REPLACE FUNCTION public.get_event(
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

-- UPD(MEMBERS)
-- DROP FUNCTION IF EXISTS create_event CASCADE;
CREATE OR REPLACE FUNCTION public.create_event(
    _name text,
    _owner uuid)
    RETURNS uuid 
    LANGUAGE 'plpgsql'

AS $BODY$
DECLARE
    new_event_id uuid;
BEGIN
    INSERT INTO events AS e (id, name, created_at, owner, image_url)
    VALUES (uuid_generate_v4(), _name, now(), _owner, '')
    RETURNING e.id INTO new_event_id;

    -- add owner to members
    INSERT INTO members (user_id, event_id)
    VALUES (_owner, new_event_id);

    RETURN new_event_id;
END;
$BODY$;