-- FUNCTION: public.get_event(uuid)

DROP FUNCTION IF EXISTS public.get_event(uuid);

CREATE OR REPLACE FUNCTION public.get_event(
	_id uuid)
    RETURNS SETOF event_type
    LANGUAGE 'plpgsql'

AS $BODY$
BEGIN
    RETURN QUERY
    SELECT events.*, users.*
    FROM events
	JOIN users ON users.id = events.owner
    WHERE events.id = _id;
END;
$BODY$;

ALTER FUNCTION public.get_event(uuid)
    OWNER TO postgres;
