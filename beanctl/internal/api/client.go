package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zeturn/beanctl/internal/auth"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
	Verbose bool
}

func New(baseURL, profile string, verbose bool) (*Client, error) {
	token, err := auth.LoadToken(profile)
	if err != nil {
		return nil, fmt.Errorf("not logged in for profile %q: %w", profile, err)
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token.AccessToken,
		HTTP:    http.DefaultClient,
		Verbose: verbose,
	}, nil
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.Do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	return c.Do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) Put(ctx context.Context, path string, body any, out any) error {
	return c.Do(ctx, http.MethodPut, path, body, out)
}

func (c *Client) Patch(ctx context.Context, path string, body any, out any) error {
	return c.Do(ctx, http.MethodPatch, path, body, out)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) Do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+"/api/v1"+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if c.Verbose {
		fmt.Printf("> %s %s\n", method, req.URL.String())
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var er struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(data, &er); err == nil && er.Error != "" {
			return fmt.Errorf("%s", er.Error)
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}
