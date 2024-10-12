-- TYPE: event type;

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
    owner_email text
);

-- FUNCTION: public.get_events();

CREATE OR REPLACE FUNCTION public.get_events()
    RETURNS SETOF event_type
    LANGUAGE 'plpgsql'
    COST 100
    VOLATILE PARALLEL UNSAFE
    ROWS 1000

AS $BODY$
BEGIN
    RETURN QUERY
    SELECT events.*, users.*
    FROM events
    JOIN users ON users.id = events.owner
    ORDER BY events.created_at DESC;
END;
$BODY$;

ALTER FUNCTION public.get_events() OWNER TO postgres;