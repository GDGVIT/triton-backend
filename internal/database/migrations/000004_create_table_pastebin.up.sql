CREATE TABLE IF NOT EXISTS public.pastebin (
	user_uuid uuid NOT NULL,
	created_at timestamptz DEFAULT now() NOT NULL,
	title text NOT NULL,
	"content" text NOT NULL,
	url_uuid uuid NOT NULL,
	extension text NOT NULL,
	CONSTRAINT pastebin_unique UNIQUE (url_uuid),
	CONSTRAINT pastebin_url_fk FOREIGN KEY (url_uuid) REFERENCES public.url(url_uuid) ON DELETE CASCADE ON UPDATE CASCADE,
	CONSTRAINT pastebin_user_fk FOREIGN KEY (user_uuid) REFERENCES public.users(uuid) ON DELETE CASCADE ON UPDATE CASCADE
);