CREATE TABLE users (
	id UUID PRIMARY KEY,
	email TEXT UNIQUE NOT NULL,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	profile_image TEXT,
	created_at TIMESTAMP DEFAULT now(),
	updated_at TIMESTAMP DEFAULT now()
);
