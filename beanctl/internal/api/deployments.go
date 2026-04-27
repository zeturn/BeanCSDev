package api

import (
	"context"
	"fmt"
)

func (c *Client) ListDeployments(ctx context.Context, project string) ([]Deployment, error) {
	id, err := c.ResolveProjectID(ctx, project)
	if err != nil {
		return nil, err
	}
	var out ListResponse[Deployment]
	return out.Data, c.Get(ctx, "/projects/"+id+"/deployments", &out)
}

func (c *Client) GetDeployment(ctx context.Context, project, deploymentID string) (*Deployment, error) {
	id, err := c.ResolveProjectID(ctx, project)
	if err != nil {
		return nil, err
	}
	var out Deployment
	return &out, c.Get(ctx, "/projects/"+id+"/deployments/"+deploymentID, &out)
}

func (c *Client) Rollback(ctx context.Context, project, deploymentID string) error {
	id, err := c.ResolveProjectID(ctx, project)
	if err != nil {
		return err
	}
	return c.Post(ctx, fmt.Sprintf("/projects/%s/deployments/%s/rollback", id, deploymentID), map[string]any{}, nil)
}
