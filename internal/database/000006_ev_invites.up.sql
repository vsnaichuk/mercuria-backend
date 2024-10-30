CREATE TABLE invites (
    id uuid DEFAULT uuidv7() PRIMARY KEY,
    event_id uuid NOT NULL,
    status VARCHAR(50) DEFAULT 'Pending' NOT NULL CHECK (status IN ('Pending', 'Accepted', 'Declined')),
    created_by uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone DEFAULT (now() + INTERVAL '24 hours') NOT NULL
);
ALTER TABLE ONLY public.invites
    ADD CONSTRAINT invites_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.invites
    ADD CONSTRAINT invites_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.events(id) ON DELETE CASCADE;
