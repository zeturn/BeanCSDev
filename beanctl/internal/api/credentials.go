package api

import "context"

func (c *Client) AddCloudflare(ctx context.Context, body map[string]any) (*Credential, error) {
	var out Credential
	return &out, c.Post(ctx, "/credentials/cloudflare", body, &out)
}

func (c *Client) ListCloudflare(ctx context.Context) ([]Credential, error) {
	var out ListResponse[Credential]
	return out.Data, c.Get(ctx, "/credentials/cloudflare", &out)
}

func (c *Client) VerifyCloudflare(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	return out, c.Get(ctx, "/credentials/cloudflare/"+id+"/verify", &out)
}

func (c *Client) AddGitHub(ctx context.Context, body map[string]any) (*Credential, error) {
	var out Credential
	return &out, c.Post(ctx, "/credentials/github", body, &out)
}

func (c *Client) ListGitHub(ctx context.Context) ([]Credential, error) {
	var out ListResponse[Credential]
	return out.Data, c.Get(ctx, "/credentials/github", &out)
}

func (c *Client) VerifyGitHub(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	return out, c.Get(ctx, "/credentials/github/"+id+"/verify", &out)
}

func (c *Client) AddBasaltPass(ctx context.Context, body map[string]any) (*Credential, error) {
	var out Credential
	return &out, c.Post(ctx, "/credentials/basaltpass", body, &out)
}

func (c *Client) ListBasaltPass(ctx context.Context) ([]Credential, error) {
	var out ListResponse[Credential]
	return out.Data, c.Get(ctx, "/credentials/basaltpass", &out)
}

func (c *Client) HealthBasaltPass(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	return out, c.Get(ctx, "/credentials/basaltpass/"+id+"/health", &out)
}

func (c *Client) ShareCredential(ctx context.Context, typ, id, user, role string) error {
	if role == "" {
		role = "user"
	}
	path := "/credentials/" + typ + "/" + id + "/share"
	return c.Post(ctx, path, map[string]any{"user_id": user, "role": role}, nil)
}

func (c *Client) DeleteCredential(ctx context.Context, typ, id string) error {
	return c.Delete(ctx, "/credentials/"+typ+"/"+id)
}
