package main

var CreateTableQuery string = `
	CREATE TABLE IF NOT EXISTS log_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message TEXT NOT NULL,
		tags TEXT,
		ts TEXT NOT NULL
	)
`

var InsertEntryQuery string = `
	INSERT INTO log_entries (message, tags, ts)
	VALUES (?, ?, ?)
`

var SelectTodayEntriesQuery string = `
	SELECT id, message, tags, ts
	FROM log_entries
	WHERE substr(ts, 1, 10) = date('now', 'localtime')
	ORDER BY ts ASC
	LIMIT 500
`
