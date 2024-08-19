CREATE TABLE IF NOT EXISTS public.users (
    uuid uuid DEFAULT uuid_generate_v4() NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    auth_type text NOT NULL,
    oauth_id text NOT NULL,
    CONSTRAINT users_unique UNIQUE (auth_type, oauth_id),
    CONSTRAINT users_pkey PRIMARY KEY (uuid)
);
