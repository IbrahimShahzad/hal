package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffc3"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#828987ff"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff95"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555"))

	focusedButton = focusedStyle.Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
)

const timeoutDuration = 10 * time.Second

type CreateUserPayload struct {
	Username string `json:"username"`
}

type CreateUserResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type RequestPayload struct {
	Message string   `json:"message"`
	Tags    []string `json:"tags,omitempty"`
}

type ResponseMsg struct {
	statusCode int
	body       string
	err        error
	userToken  string
}

type AppMode int

const (
	ModeCreateUser AppMode = iota
	ModePostMessage
)

type model struct {
	mode       AppMode
	focusIndex int
	inputs     []textinput.Model
	baseURL    string
	token      string
	response   *ResponseMsg
	submitting bool
}

func getTokenFilePath(username string) string {
	homeDir, _ := os.UserHomeDir()
	username = strings.ToUpper(strings.TrimSpace(username))
	return filepath.Join(homeDir, ".hal", fmt.Sprintf("%s.token", username))
}

func saveToken(username, token string) error {
	filePath := getTokenFilePath(username)
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(token), 0600)
}

func loadToken(username string) string {
	filePath := getTokenFilePath(username)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func initialModel(baseURL string) model {
	m := model{
		mode:    ModeCreateUser,
		baseURL: baseURL,
		token:   "",
	}

	m.setupInputs()
	return m
}

func (m *model) setupInputs() {
	switch m.mode {
	case ModeCreateUser:
		m.inputs = make([]textinput.Model, 1)
		t := textinput.New()
		t.Placeholder = "Enter username..."
		t.Focus()
		t.PromptStyle = focusedStyle
		t.TextStyle = focusedStyle
		t.CharLimit = 50
		t.Width = 50
		m.inputs[0] = t
	case ModePostMessage:
		m.inputs = make([]textinput.Model, 2)
		for i := range m.inputs {
			t := textinput.New()
			t.Cursor.Style = cursorStyle

			switch i {
			case 0:
				t.Placeholder = "Enter your message..."
				t.Focus()
				t.PromptStyle = focusedStyle
				t.TextStyle = focusedStyle
				t.CharLimit = 500
				t.Width = 50
			case 1:
				t.Placeholder = "Enter tags (comma-separated, optional)"
				t.CharLimit = 200
				t.Width = 50
			}
			m.inputs[i] = t
		}
	}
	m.focusIndex = 0
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "backspace":
			if m.mode == ModePostMessage {
				if m.focusIndex < len(m.inputs) && m.inputs[m.focusIndex].Value() == "" {
					m.mode = ModeCreateUser
					m.token = ""
					m.setupInputs()
					return m, nil
				}
				if m.focusIndex >= len(m.inputs) {
					m.mode = ModeCreateUser
					m.token = ""
					m.setupInputs()
					return m, nil
				}
			}

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" && m.focusIndex == len(m.inputs) {
				if m.mode == ModeCreateUser && len(m.inputs) > 0 && m.inputs[0].Value() != "" {
					username := m.inputs[0].Value()
					token := loadToken(username)
					if token != "" {
						m.mode = ModePostMessage
						m.token = token
						m.setupInputs()
						return m, textinput.Blink
					} else {
						m.submitting = true
						return m, createUser(m.baseURL, username)
					}
				} else if m.mode == ModePostMessage && len(m.inputs) > 0 && m.inputs[0].Value() != "" {
					m.submitting = true
					return m, submitMessage(m.baseURL, m.inputs[0].Value(), m.inputs[1].Value(), m.token)
				}
				return m, nil
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}

	case ResponseMsg:
		m.submitting = false
		m.response = &msg
		if msg.err == nil {
			if m.mode == ModeCreateUser && msg.userToken != "" {
				username := m.inputs[0].Value()
				if err := saveToken(username, msg.userToken); err != nil {
					m.response.err = fmt.Errorf("failed to save token: %w", err)
				} else {
					m.token = msg.userToken
					m.mode = ModePostMessage
					m.setupInputs()
					return m, textinput.Blink
				}
			}
		} else if msg.err != nil && m.mode == ModeCreateUser {
			if strings.Contains(msg.err.Error(), "username already exists") ||
				strings.Contains(msg.err.Error(), "already exists") {
				username := m.inputs[0].Value()
				token := loadToken(username)
				if token != "" {
					m.token = token
					m.mode = ModePostMessage
					m.response = nil
					m.setupInputs()
					return m, textinput.Blink
				}
				m.response.err = fmt.Errorf("user %s exists but no saved token found", username)
			}
		}

		if msg.err == nil {
			if len(m.inputs) > 0 {
				m.inputs[0].SetValue("")
			}
			if len(m.inputs) > 1 {
				m.inputs[1].SetValue("")
			}
			m.focusIndex = 0
			if len(m.inputs) > 0 {
				m.inputs[0].Focus()
				m.inputs[0].PromptStyle = focusedStyle
				m.inputs[0].TextStyle = focusedStyle
			}
		}
		return m, nil
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(focusedStyle.Render("HAL: Status Report, Dave!") + "\n\n")

	switch m.mode {
	case ModeCreateUser:
		b.WriteString(helpStyle.Render("Enter Username:") + "\n\n")
		if len(m.inputs) > 0 {
			b.WriteString(helpStyle.Render("Username:") + "\n")
			b.WriteString(m.inputs[0].View())
			b.WriteString("\n\n")
		}

		button := &blurredButton
		if m.focusIndex == len(m.inputs) {
			button = &focusedButton
		}
		fmt.Fprintf(&b, "%s\n\n", *button)

	case ModePostMessage:
		b.WriteString(helpStyle.Render("Post Message:") + "\n\n")
		if len(m.inputs) > 0 {
			b.WriteString(helpStyle.Render("Message:") + "\n")
			b.WriteString(m.inputs[0].View())
			b.WriteString("\n\n")
		}
		if len(m.inputs) > 1 {
			b.WriteString(helpStyle.Render("Tags:") + "\n")
			b.WriteString(m.inputs[1].View())
			b.WriteString("\n\n")
		}

		button := &blurredButton
		if m.focusIndex == len(m.inputs) {
			button = &focusedButton
		}
		fmt.Fprintf(&b, "%s\n\n", *button)
	}

	if m.submitting {
		b.WriteString(helpStyle.Render("Submitting...") + "\n\n")
	}

	if m.response != nil {
		if m.response.err != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.response.err)) + "\n\n")
		} else {
			b.WriteString(successStyle.Render(fmt.Sprintf("Success! Status: %d", m.response.statusCode)) + "\n")
			if m.response.body != "" {
				b.WriteString(helpStyle.Render(fmt.Sprintf("Response: %s", m.response.body)) + "\n")
			}
			if m.response.userToken != "" {
				b.WriteString(successStyle.Render("Token saved locally!") + "\n")
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(helpStyle.Render(fmt.Sprintf("Endpoint: %s", m.baseURL)) + "\n")
	if m.token != "" {
		b.WriteString(helpStyle.Render("Auth: Token configured") + "\n")
	} else {
		b.WriteString(errorStyle.Render("Auth: No token") + "\n")
	}

	switch m.mode {
	case ModeCreateUser:
		b.WriteString(helpStyle.Render("tab/shift+tab: navigate • enter: submit • esc: quit") + "\n")
	case ModePostMessage:
		b.WriteString(helpStyle.Render("backspace: back to username • tab/shift+tab: navigate • enter: submit • esc: quit") + "\n")
	}

	return b.String()
}

func createUser(baseURL, username string) tea.Cmd {
	return func() tea.Msg {
		payload := CreateUserPayload{Username: username}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return ResponseMsg{err: fmt.Errorf("failed to marshal JSON: %w", err)}
		}

		client := &http.Client{Timeout: timeoutDuration}
		url := fmt.Sprintf("%s/users", baseURL)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return ResponseMsg{err: fmt.Errorf("failed to create request: %w", err)}
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return ResponseMsg{err: fmt.Errorf("failed to send request: %w", err)}
		}
		defer resp.Body.Close() // nolint:errcheck

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ResponseMsg{
				statusCode: resp.StatusCode,
				err:        fmt.Errorf("failed to read response: %w", err),
			}
		}

		if resp.StatusCode >= 400 {
			return ResponseMsg{
				statusCode: resp.StatusCode,
				body:       string(body),
				err:        fmt.Errorf("server error: %s", string(body)),
			}
		}

		var userResp CreateUserResponse
		if err := json.Unmarshal(body, &userResp); err == nil {
			return ResponseMsg{
				statusCode: resp.StatusCode,
				body:       string(body),
				userToken:  userResp.Token,
			}
		}

		return ResponseMsg{
			statusCode: resp.StatusCode,
			body:       string(body),
		}
	}
}

func submitMessage(baseURL, message, tagsStr, token string) tea.Cmd {
	return func() tea.Msg {
		var tags []string
		if tagsStr != "" {
			for tag := range strings.SplitSeq(tagsStr, ",") {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		}

		payload := RequestPayload{
			Message: message,
			Tags:    tags,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return ResponseMsg{err: fmt.Errorf("failed to marshal JSON: %w", err)}
		}

		client := &http.Client{Timeout: timeoutDuration}
		url := fmt.Sprintf("%s/update", baseURL)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return ResponseMsg{err: fmt.Errorf("failed to create request: %w", err)}
		}

		req.Header.Set("Content-Type", "application/json")

		if token != "" {
			req.Header.Set("X-Auth-Token", token)
		}

		resp, err := client.Do(req)
		if err != nil {
			return ResponseMsg{err: fmt.Errorf("failed to send request: %w", err)}
		}
		defer resp.Body.Close() // nolint:errcheck

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ResponseMsg{
				statusCode: resp.StatusCode,
				err:        fmt.Errorf("failed to read response: %w", err),
			}
		}

		if resp.StatusCode >= 400 {
			return ResponseMsg{
				statusCode: resp.StatusCode,
				body:       string(body),
				err:        fmt.Errorf("server error: %s", string(body)),
			}
		}

		return ResponseMsg{
			statusCode: resp.StatusCode,
			body:       string(body),
		}
	}
}

func main() {
	addr := flag.String("addr", "localhost:8080", "Server address (host:port)")
	flag.Parse()

	baseURL := fmt.Sprintf("http://%s", *addr)

	p := tea.NewProgram(initialModel(baseURL))
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %s\n", err)
	}
}
