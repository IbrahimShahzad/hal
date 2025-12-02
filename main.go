package main

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
)

// Must is a helper function to handle errors.
// In case of an error, log it and exit the program.
// It returns the value if no error occurred.
func Must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	db := Must(sql.Open("sqlite3", "./worklog.db"))

	Must(db.Exec(CreateUsersTableQuery))
	Must(db.Exec(CreateTableQuery))

	s := NewServer(db)

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/audio/", http.StripPrefix("/audio/", http.FileServer(http.Dir("audio"))))

	mux.HandleFunc("POST /users", s.handleCreateUser)

	mux.HandleFunc("/initial/", s.handleInitial)
	mux.HandleFunc("/initial", s.handleInitial)
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/update", s.handlePost)
	mux.HandleFunc("GET /user/{username}", s.handleUserIndex)
	mux.HandleFunc("/", s.handleIndex)

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		srv.Shutdown(context.Background()) // nolint:errcheck
	}()

	log.Println("Listening on", *addr)
	log.Println("Create users with: POST /users {\"username\": \"alex\"}")
	log.Println("View user logs at: /user/{username}")
	log.Fatal(srv.ListenAndServe())
}
