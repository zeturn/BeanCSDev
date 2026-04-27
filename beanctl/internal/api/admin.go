package api

import "context"

func (c *Client) Overview(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	return out, c.Get(ctx, "/admin/overview", &out)
}

func (c *Client) Nodes(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	return out, c.Get(ctx, "/admin/nodes", &out)
}

func (c *Client) Quotas(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	return out, c.Get(ctx, "/admin/quotas", &out)
}

func (c *Client) SetQuota(ctx context.Context, team string, body map[string]any) (map[string]any, error) {
	var out map[string]any
	return out, c.Patch(ctx, "/admin/quotas/"+team, body, &out)
}
