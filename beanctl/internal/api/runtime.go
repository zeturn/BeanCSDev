package api

import (
	"context"
	"fmt"
)

func (c *Client) Status(ctx context.Context, project string) (map[string]any, error) {
	id, err := c.ResolveProjectID(ctx, project)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	return out, c.Get(ctx, "/projects/"+id+"/status", &out)
}

func (c *Client) Logs(ctx context.Context, project string, tail int) (map[string]string, error) {
	id, err := c.ResolveProjectID(ctx, project)
	if err != nil {
		return nil, err
	}
	var out map[string]string
	return out, c.Get(ctx, fmt.Sprintf("/projects/%s/logs?tail=%d", id, tail), &out)
}

func (c *Client) Restart(ctx context.Context, project string) error {
	id, err := c.ResolveProjectID(ctx, project)
	if err != nil {
		return err
	}
	return c.Post(ctx, "/projects/"+id+"/restart", map[string]any{}, nil)
}

func (c *Client) Scale(ctx context.Context, project string, replicas int) error {
	id, err := c.ResolveProjectID(ctx, project)
	if err != nil {
		return err
	}
	return c.Post(ctx, "/projects/"+id+"/scale", map[string]any{"replicas": replicas}, nil)
}
