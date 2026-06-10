// Package api is the HTTP client the TUI uses to talk to the inkwell API.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

type EntryMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type Entry struct {
	EntryMeta
	Body string `json:"body"`
}

func (c *Client) ListEntries() ([]EntryMeta, error) {
	var out []EntryMeta
	if err := c.do(http.MethodGet, "/entries", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetEntry(id string) (*Entry, error) {
	var out Entry
	if err := c.do(http.MethodGet, "/entries/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateEntry(title, body string) (*EntryMeta, error) {
	var out EntryMeta
	if err := c.do(http.MethodPost, "/entries", map[string]string{"title": title, "body": body}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteEntry(id string) error {
	return c.do(http.MethodDelete, "/entries/"+id, nil, nil)
}

var ErrUnauthorized = errors.New("unauthorized")

func (c *Client) do(method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode == 401 {
		return ErrUnauthorized
	}
	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("%d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	if res.StatusCode == 204 || out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
