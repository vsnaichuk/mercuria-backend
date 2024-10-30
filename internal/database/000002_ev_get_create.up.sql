-- FUNCTION: public.get_event(uuid);

CREATE OR REPLACE FUNCTION public.get_event(
    _id uuid)
    RETURNS TABLE(id uuid, name text, created_at timestamp with time zone, owner uuid, image_url text)
    LANGUAGE 'plpgsql'
    COST 100
    VOLATILE PARALLEL UNSAFE
    ROWS 1000

AS $BODY$
BEGIN
    RETURN QUERY
    SELECT *
    FROM events e
    WHERE e.id = _id;
END;
$BODY$;

ALTER FUNCTION public.get_event(uuid) OWNER TO postgres;

-- FUNCTION: public.create_event(text, uuid)

CREATE OR REPLACE FUNCTION public.create_event(
    _name text,
    _owner uuid)
    RETURNS TABLE(id uuid)
    LANGUAGE 'plpgsql'
    COST 100
    VOLATILE PARALLEL UNSAFE
    ROWS 1000

AS $BODY$
BEGIN
    RETURN QUERY
    INSERT INTO events AS e (id, name, created_at, owner, image_url)
    VALUES (uuidv7(), _name, now(), _owner, '')
    RETURNING e.id;
END;
$BODY$;

ALTER FUNCTION public.create_event(text, uuid) OWNER TO postgres;
