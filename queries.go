package main

import (
	"crypto/rand"
	"encoding/hex"
)

func generateToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// FIXME: handle this properly
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

var CreateUsersTableQuery string = `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		token TEXT UNIQUE NOT NULL,
		created_at TEXT NOT NULL
	)
`

var CreateTableQuery string = `
	CREATE TABLE IF NOT EXISTS log_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		message TEXT NOT NULL,
		tags TEXT,
		ts TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users (id)
	)
`

var CreateUserQuery string = `
	INSERT INTO users (username, token, created_at)
	VALUES (?, ?, ?)
`

var GetUserByTokenQuery string = `
	SELECT id, username FROM users WHERE token = ?
`

var GetUserByUsernameQuery string = `
	SELECT id, username FROM users WHERE username = ?
`

var InsertEntryQuery string = `
	INSERT INTO log_entries (user_id, message, tags, ts)
	VALUES (?, ?, ?, ?)
`

var SelectTodayEntriesQuery string = `
	SELECT le.id, u.username, le.message, le.tags, le.ts
	FROM log_entries le
	LEFT JOIN users u ON le.user_id = u.id
	WHERE substr(le.ts, 1, 10) = date('now', 'localtime')
	ORDER BY le.ts ASC
	LIMIT 500
`

var SelectTodayEntriesByUserQuery string = `
	SELECT le.id, u.username, le.message, le.tags, le.ts
	FROM log_entries le
	JOIN users u ON le.user_id = u.id
	WHERE u.username = ? AND substr(le.ts, 1, 10) = date('now', 'localtime')
	ORDER BY le.ts ASC
	LIMIT 500
`
