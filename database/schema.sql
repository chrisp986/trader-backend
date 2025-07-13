-- Enable foreign key constraints
PRAGMA foreign_keys = ON;


CREATE TABLE users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
			
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
		


