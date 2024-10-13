DROP TABLE likes;
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

DROP TYPE IF EXISTS event_like_type CASCADE;
CREATE TYPE event_like_type AS (
    id bigint,
    user_id uuid,
    event_id uuid,
    created_at timestamp with time zone
);

DROP TYPE IF EXISTS event_type CASCADE;
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
    -- UPD
    like_id bigint,
    like_user_id uuid,
    like_event_id uuid,
    like_created_at timestamp with time zone
);

-- UPD FUNCTION: public.get_events();
CREATE OR REPLACE FUNCTION public.get_events()
    RETURNS SETOF event_type
    LANGUAGE 'plpgsql'
    COST 100
    VOLATILE PARALLEL UNSAFE
    ROWS 1000

AS $BODY$
BEGIN
    RETURN QUERY
    SELECT events.*, users.*, likes.*
    FROM events
    JOIN users ON users.id = events.owner
    LEFT JOIN likes ON likes.event_id = events.id
    ORDER BY events.created_at DESC;
END;
$BODY$;

-- UPD FUNCTION: public.get_event();
CREATE OR REPLACE FUNCTION public.get_event(
    _id uuid)
    RETURNS SETOF event_type
    LANGUAGE 'plpgsql'

AS $BODY$
BEGIN
    RETURN QUERY
    SELECT events.*, users.*, likes.*
    FROM events
    JOIN users ON users.id = events.owner
    LEFT JOIN likes ON likes.event_id = events.id
    WHERE events.id = _id;
END;
$BODY$;