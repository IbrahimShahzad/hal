package main

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type Update struct {
	ID        int64    `json:"id"`
	Message   string   `json:"message"`
	Tags      []string `json:"tags,omitempty"`
	Timestamp string   `json:"timestamp"`
}

type Server struct {
	db        *sql.DB
	clientsMu sync.RWMutex
	clients   map[chan Update]struct{}
	broadcast chan Update
	token     string
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func NewServer(db *sql.DB, token string) *Server {
	s := &Server{
		db:        db,
		clients:   make(map[chan Update]struct{}),
		broadcast: make(chan Update, 32),
		token:     token,
	}
	go s.runBroadcaster()
	return s
}

func (s *Server) runBroadcaster() {
	for u := range s.broadcast {
		s.clientsMu.RLock()
		for ch := range s.clients {
			select {
			case ch <- u:
			default:
			}
		}
		s.clientsMu.RUnlock()
	}
}

func (s *Server) addClient(ch chan Update) {
	s.clientsMu.Lock()
	s.clients[ch] = struct{}{}
	s.clientsMu.Unlock()
}

func (s *Server) removeClient(ch chan Update) {
	s.clientsMu.Lock()
	if _, ok := s.clients[ch]; ok {
		delete(s.clients, ch)
		close(ch)
	}
	s.clientsMu.Unlock()
}

func (s *Server) insertUpdate(u *Update) error {
	res, err := s.db.Exec(`
		INSERT INTO log_entries (message, tags, ts)
		VALUES (?, ?, ?)
	`, u.Message, strings.Join(u.Tags, ","), u.Timestamp)
	if err != nil {
		return err
	}

	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./simple.html")
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	clientCh := make(chan Update, 16)
	s.addClient(clientCh)
	defer s.removeClient(clientCh)

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case u := <-clientCh:
			b, _ := json.Marshal(u)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

func (s *Server) handleInitial(w http.ResponseWriter, r *http.Request) {
	// we display in descending order on frontend, so we need to keep the
	// order opposite here
	rows, err := s.db.Query(`
		SELECT id, message, tags, ts
		FROM log_entries
		WHERE substr(ts, 1, 10) = date('now', 'localtime');
		ORDER BY ts ASC
		LIMIT 500
	`)
	must(err)
	defer rows.Close()

	list := []Update{}

	for rows.Next() {
		var (
			id   int64
			msg  string
			tags string
			ts   string
		)
		rows.Scan(&id, &msg, &tags, &ts)
		list = append(list, Update{
			ID:      id,
			Message: msg,
			Tags: func() []string {
				if tags == "" {
					return nil
				}
				return strings.Split(tags, ",")
			}(),
			Timestamp: ts,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if r.Header.Get("X-Auth-Token") != s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var in struct {
		Message string   `json:"message"`
		Tags    []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if in.Message == "" {
		http.Error(w, "empty message", http.StatusBadRequest)
		return
	}

	u := Update{
		Message:   in.Message,
		Tags:      in.Tags,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	must(s.insertUpdate(&u))

	select {
	case s.broadcast <- u:
	default:
		go func() { s.broadcast <- u }()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	token := flag.String("token", "", "auth token")
	flag.Parse()

	db, err := sql.Open("sqlite3", "./worklog.db")
	must(err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS log_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message TEXT NOT NULL,
			tags TEXT,
			ts TEXT NOT NULL
		)
	`)
	must(err)

	s := NewServer(db, *token)

	mux := http.NewServeMux()
	mux.HandleFunc("/initial", s.handleInitial)
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/update", s.handlePost)
	mux.HandleFunc("/", s.handleIndex)

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		srv.Shutdown(context.Background())
	}()

	log.Println("Listening on", *addr)
	log.Fatal(srv.ListenAndServe())
}
