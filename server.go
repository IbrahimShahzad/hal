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

// processTags takes a slice of tags and normalizes them.
// Converts to uppercase, Replaces spaces with underscores, Trims whitespace
func processTags(tags []string) []string {
	if len(tags) == 0 {
		return tags
	}

	processed := make([]string, len(tags))
	for i, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = strings.ToUpper(tag)
		tag = strings.ReplaceAll(tag, " ", "_")
		processed[i] = tag
	}
	return processed
}

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token,omitempty"`
}

type Update struct {
	ID        int64    `json:"id"`
	Username  string   `json:"username,omitempty"`
	Message   string   `json:"message"`
	Tags      []string `json:"tags,omitempty"`
	Timestamp string   `json:"timestamp"`
}

type Server struct {
	db        *sql.DB
	clientsMu sync.RWMutex
	clients   map[chan Update]struct{}
	broadcast chan Update
}

func NewServer(db *sql.DB) *Server {
	s := &Server{
		db:        db,
		clients:   make(map[chan Update]struct{}),
		broadcast: make(chan Update, 32),
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

func (s *Server) getUserByToken(token string) (*User, error) {
	var user User
	err := s.db.QueryRow(GetUserByTokenQuery, token).Scan(&user.ID, &user.Username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Server) createUser(username string) (*User, error) {
	username = strings.ToUpper(strings.TrimSpace(username))

	token := generateToken()
	timestamp := time.Now().Format(time.RFC3339)

	res, err := s.db.Exec(CreateUserQuery, username, token, timestamp)
	if err != nil {
		return nil, err
	}

	id, _ := res.LastInsertId()
	return &User{
		ID:       id,
		Username: username,
		Token:    token,
	}, nil
}

func (s *Server) insertUpdate(u *Update, userID int64) error {
	res, err := s.db.Exec(InsertEntryQuery, userID, u.Message, strings.Join(u.Tags, ","), u.Timestamp)
	if err != nil {
		return err
	}

	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if in.Username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	user, err := s.createUser(in.Username)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user) // nolint:errcheck
}

func (s *Server) handleUserIndex(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/user/")
	username := strings.Trim(path, "/")

	username = strings.ToUpper(strings.TrimSpace(username))

	var id int64
	var actualUsername string
	err := s.db.QueryRow(GetUserByUsernameQuery, username).Scan(&id, &actualUsername)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, "./index.html")
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
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
	path := strings.TrimPrefix(r.URL.Path, "/initial")
	username := strings.Trim(path, "/")

	if username != "" {
		username = strings.ToUpper(strings.TrimSpace(username))
	}

	var rows *sql.Rows
	var err error

	if username != "" {
		rows, err = s.db.Query(SelectTodayEntriesByUserQuery, username)
	} else {
		rows, err = s.db.Query(SelectTodayEntriesQuery)
	}

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close() // nolint:errcheck

	list := []Update{}

	for rows.Next() {
		var (
			id       int64
			username sql.NullString
			msg      string
			tags     string
			ts       string
		)
		rows.Scan(&id, &username, &msg, &tags, &ts) // nolint:errcheck

		update := Update{
			ID:      id,
			Message: msg,
			Tags: func() []string {
				if tags == "" {
					return nil
				}
				return strings.Split(tags, ",")
			}(),
			Timestamp: ts,
		}

		if username.Valid {
			update.Username = username.String
		}

		list = append(list, update)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list) // nolint:errcheck
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Auth-Token")
	if token == "" {
		http.Error(w, "authentication token required", http.StatusUnauthorized)
		return
	}

	user, err := s.getUserByToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
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
		Username:  user.Username,
		Message:   in.Message,
		Tags:      processTags(in.Tags),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := s.insertUpdate(&u, user.ID); err != nil {
		http.Error(w, "failed to insert update", http.StatusInternalServerError)
		return
	}

	select {
	case s.broadcast <- u:
	default:
		go func() { s.broadcast <- u }()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u) // nolint:errcheck
}
