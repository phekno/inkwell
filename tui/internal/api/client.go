// Package api is the HTTP client the TUI uses to talk to the inkwell API.
package api

import (
	"errors"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

type Entry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) ListEntries() ([]Entry, error) {
	return nil, errors.New("not implemented")
}

func (c *Client) CreateEntry(title, body string) (*Entry, error) {
	return nil, errors.New("not implemented")
}
