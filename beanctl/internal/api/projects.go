package api

import (
	"context"
	"fmt"
)

func (c *Client) CreateProject(ctx context.Context, body map[string]any) (*Project, error) {
	var out Project
	return &out, c.Post(ctx, "/projects", body, &out)
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var out ListResponse[Project]
	return out.Data, c.Get(ctx, "/projects", &out)
}

func (c *Client) GetProject(ctx context.Context, ref string) (*Project, error) {
	id, err := c.ResolveProjectID(ctx, ref)
	if err != nil {
		return nil, err
	}
	var out Project
	return &out, c.Get(ctx, "/projects/"+id, &out)
}

func (c *Client) UpdateProject(ctx context.Context, ref string, body map[string]any) (*Project, error) {
	id, err := c.ResolveProjectID(ctx, ref)
	if err != nil {
		return nil, err
	}
	var out Project
	return &out, c.Patch(ctx, "/projects/"+id, body, &out)
}

func (c *Client) DeleteProject(ctx context.Context, ref string) error {
	id, err := c.ResolveProjectID(ctx, ref)
	if err != nil {
		return err
	}
	return c.Delete(ctx, "/projects/"+id)
}

func (c *Client) ResolveProjectID(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("project is required")
	}
	if isDigits(ref) {
		return ref, nil
	}
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range projects {
		if p.Name == ref {
			return fmt.Sprint(p.ID), nil
		}
	}
	return "", fmt.Errorf("project %q not found", ref)
}

func (c *Client) GetEnv(ctx context.Context, ref string) (map[string]string, error) {
	id, err := c.ResolveProjectID(ctx, ref)
	if err != nil {
		return nil, err
	}
	var out struct {
		Data map[string]string `json:"data"`
	}
	return out.Data, c.Get(ctx, "/projects/"+id+"/env", &out)
}

func (c *Client) PutEnv(ctx context.Context, ref string, env map[string]string) error {
	id, err := c.ResolveProjectID(ctx, ref)
	if err != nil {
		return err
	}
	return c.Put(ctx, "/projects/"+id+"/env", env, nil)
}

func isDigits(v string) bool {
	for _, r := range v {
		if r < '0' || r > '9' {
			return false
		}
	}
	return v != ""
}
