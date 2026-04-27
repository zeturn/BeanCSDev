# BeanCS GitHub Actions k3s Deployment

This repository can deploy BeanCS from GitHub Actions to a k3s cluster.

## Flow

- Push to `main` or `deploy`: run tests, build the controller image, push to GHCR.
- Push a `v*` tag: verify the tag commit belongs to `origin/deploy`, then deploy to k3s.

## Required GitHub Secrets

- `KUBE_CONFIG`: kubeconfig content for the target k3s cluster.
- `BEANCS_POSTGRES_PASSWORD`: password for the bundled Postgres StatefulSet.
- `BP_MGMT_BASE_URL`: BasaltPass base URL.
- `BP_MGMT_CLIENT_ID`: BasaltPass management OAuth client ID.
- `BP_MGMT_CLIENT_SECRET`: BasaltPass management OAuth client secret.
- `BP_BROWSER_AUTH_URL`: BasaltPass browser login base URL.
- `BP_BROWSER_CLIENT_ID`: BasaltPass browser OAuth client ID.
- `BP_BROWSER_CLIENT_SECRET`: BasaltPass browser OAuth client secret.
- `BEANCS_WEBHOOK_SECRET`: webhook verification secret.
- `BEANCS_ENCRYPTION_KEY`: 64 hex characters.

Optional:

- `BEANCS_DATABASE_URL`: external database URL. If omitted, the workflow points BeanCS at the bundled Postgres service.
- `GHCR_PAT`: package token for image pulls when `GITHUB_TOKEN` is not enough.

## Required GitHub Variables

- `INGRESS_IP`: public Traefik ingress IP.
- `BEANCS_PUBLIC_HOST`: public BeanCS dashboard host.
- `BEANCS_WEBHOOK_HOST`: public webhook host.
- `BEANCS_TAILSCALE_HOST`: Tailscale-only host, defaults to `beancs-controller` in the workflow.

## Release

```bash
git checkout deploy
git pull --ff-only origin deploy
git tag v1.0.0
git push origin deploy --tags
```

BeanCS will deploy to the `beancs-system` namespace and reconcile its own Service and Ingress resources on startup.
