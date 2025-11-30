package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
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
	res, err := s.db.Exec(InsertEntryQuery, u.Message, strings.Join(u.Tags, ","), u.Timestamp)
	if err != nil {
		return err
	}

	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./index.html")
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
			fmt.Fprintf(w, "data: %s\n\n", b) // nolint:errcheck
			flusher.Flush()
		}
	}
}

func (s *Server) handleInitial(w http.ResponseWriter, r *http.Request) {
	// FIXME: the app should not fail here or should it ???
	// Hmmm!
	// should continue with empty list?
	// NOTE: we display in descending order on frontend, so we need to keep the
	// order opposite here so select query orders by ts ASC
	rows := Must(s.db.Query(SelectTodayEntriesQuery))
	defer rows.Close() // nolint:errcheck

	list := []Update{}

	for rows.Next() {
		var (
			id   int64
			msg  string
			tags string
			ts   string
		)
		rows.Scan(&id, &msg, &tags, &ts) // nolint:errcheck
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
	json.NewEncoder(w).Encode(list) // nolint:errcheck
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

	// FIXME: may be should not fail here, will updated later
	// send 500 error instead
	Must(0, s.insertUpdate(&u))

	select {
	case s.broadcast <- u:
	default:
		go func() { s.broadcast <- u }()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u) // nolint:errcheck
}
