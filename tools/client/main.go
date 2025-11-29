package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Need to send the following type of request from the
// client
// curl -X POST http://localhost:8080/update \
//   -H "Content-Type: application/json" \
//   -H "X-Auth-Token: myuperduperstrongandmightypassword" \
//   -d '{"message":"some more testing here"}'

const TIMEOUT = 5 * time.Second
const PATH_UPDATE = "/update"
const AUTH_TOKEN = "AUTH_TOKEN"
const APP_NAME = "worklog-client/0.1"

// http headers
const AUTH_HEADER = "X-Auth-Token"
const CONTENT_TYPE_HEADER = "Content-Type"
const CONTENT_TYPE_JSON = "application/json"
const USER_AGENT_HEADER = "User-Agent"
const CONTENT_LENGTH_HEADER = "Content-Length"

var (
	ErrTokenMissing = errors.New("X-Auth-Token must be provided via -token flag or AUTH_TOKEN environment variable")
	ErrMessageEmpty = errors.New("message cannot be empty, must be provided via -m flag")
)

type Message struct {
	Message string   `json:"message"`
	Tags    []string `json:"tags,omitempty"`
}

func generateURL(addr, path string) string {
	return "http://" + addr + path
}

func splitTags(tags string) []string {
	var result []string
	for tag := range bytes.SplitSeq([]byte(tags), []byte(",")) {
		trimmed := bytes.TrimSpace(tag)
		if len(trimmed) > 0 {
			result = append(result, string(trimmed))
		}
	}
	return result
}

func encodeJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func Must[T any](val T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return val
}

func setDefaultHeaders(req *http.Request, token string, contentLength int) {
	req.Header.Set(AUTH_HEADER, token)
	req.Header.Set(CONTENT_TYPE_HEADER, CONTENT_TYPE_JSON)
	req.Header.Set(USER_AGENT_HEADER, APP_NAME)
	req.Header.Set(
		CONTENT_LENGTH_HEADER,
		fmt.Sprintf("%d", contentLength),
	)
}

// validateInput checks command line flags and environment variables
// returns addr, token, message, tags
// exits the program if validation fails
func validateInput() (string, string, string, []string) {
	addr := flag.String("addr", ":8080", "HTTP server address")
	token := flag.String("token", "", "X-Auth-Token")
	message := flag.String("m", "", "Message to send")
	tags := flag.String("t", "", "Comma-separated list of tags")
	var tagsArr []string = nil

	flag.Parse()

	if *token == "" {
		*token = os.Getenv(AUTH_TOKEN)
		if *token == "" {
			log.Fatal(ErrTokenMissing)
		}
	}

	if *message == "" {
		log.Fatal(ErrMessageEmpty)
	}

	if *tags != "" {
		tagsArr = splitTags(*tags)
	}

	return *addr, *token, *message, tagsArr

}

func main() {

	addr, token, message, tags := validateInput()

	msg := &Message{
		Message: message,
		Tags:    tags,
	}

	body := Must(encodeJSON(msg))

	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	request := Must(http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		generateURL(addr, PATH_UPDATE),
		bytes.NewReader(body),
	))
	setDefaultHeaders(request, token, len(body))

	client := &http.Client{}
	response := Must(client.Do(request))
	defer response.Body.Close()

	rspBody := Must(io.ReadAll(response.Body))
	log.Printf("Response status: %s, %s", response.Status, rspBody)
}
