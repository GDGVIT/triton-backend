CREATE TABLE IF NOT EXISTS public.url (
	url_uuid uuid DEFAULT uuid_generate_v4() NOT NULL,
	url_name public.citext NOT NULL,
	CONSTRAINT url_pk PRIMARY KEY (url_uuid),
	CONSTRAINT url_unique UNIQUE (url_name)
);