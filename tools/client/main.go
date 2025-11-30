package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

// Default URL - can be changed via environment variable or flag POST_URL
const defaultURL = "http://localhost:8080/update"
const timeoutDuration = 10 * time.Second
const RequestMethod = "POST"

type RequestPayload struct {
	Message string   `json:"message"`
	Tags    []string `json:"tags,omitempty"`
}

type ResponseMsg struct {
	statusCode int
	body       string
	err        error
}

type model struct {
	focusIndex int
	inputs     []textinput.Model
	url        string
	token      string
	response   *ResponseMsg
	submitting bool
}

func initialModel() model {
	m := model{
		inputs: make([]textinput.Model, 2),
		url:    getURL(),
		token:  getToken(),
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
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

	return m
}

func getURL() string {
	if url := os.Getenv("POST_URL"); url != "" {
		return url
	}
	return defaultURL
}

func getToken() string {
	return os.Getenv("AUTH_TOKEN")
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

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" && m.focusIndex == len(m.inputs) {
				if m.inputs[0].Value() == "" {
					return m, nil
				}
				m.submitting = true
				return m, submitForm(m.url, m.inputs[0].Value(), m.inputs[1].Value(), m.token)
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
			// Reset form on success
			m.inputs[0].SetValue("")
			m.inputs[1].SetValue("")
			m.focusIndex = 0
			m.inputs[0].Focus()
			m.inputs[0].PromptStyle = focusedStyle
			m.inputs[0].TextStyle = focusedStyle
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

	b.WriteString(helpStyle.Render("Message:") + "\n")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Tags:") + "\n")
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "%s\n\n", *button)

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
			b.WriteString("\n")
		}
	}

	b.WriteString(helpStyle.Render(fmt.Sprintf("Endpoint: %s", m.url)) + "\n")
	if m.token != "" {
		b.WriteString(helpStyle.Render("Auth: Token configured") + "\n")
	} else {
		b.WriteString(errorStyle.Render("Auth: No token") + "\n")
	}

	b.WriteString(helpStyle.Render("tab/shift+tab: navigate • enter: submit • esc: quit") + "\n")
	return b.String()
}

func submitForm(url, message, tagsStr, token string) tea.Cmd {
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

		client := &http.Client{
			Timeout: timeoutDuration,
		}

		req, err := http.NewRequest(RequestMethod, url, bytes.NewBuffer(jsonData))
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
			err:        nil,
		}
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %s\n", err)
	}
}
