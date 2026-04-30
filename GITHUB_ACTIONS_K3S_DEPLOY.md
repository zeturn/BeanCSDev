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
- `BP_BROWSER_AUTH_URL`: BasaltPass browser login base URL. Either the root host or `/api/v1` base is accepted; BeanCS normalizes OAuth calls to `/api/v1/oauth/*`.
- `BP_BROWSER_CLIENT_ID`: BasaltPass browser OAuth client ID.
- `BP_BROWSER_CLIENT_SECRET`: BasaltPass browser OAuth client secret.
- `BEANCS_WEBHOOK_SECRET`: webhook verification secret.
- `BEANCS_ENCRYPTION_KEY`: 64 hex characters.

Optional:

- `BEANCS_DATABASE_URL`: external database URL. If omitted, the workflow points BeanCS at the bundled Postgres service.
- `GHCR_PAT`: package token for image pulls when `GITHUB_TOKEN` is not enough.
- `BEANCS_PUBLIC_INGRESS_NAMESPACES`: namespaces that may route public traffic to BeanCS-managed pods, defaults to `kube-system,traefik`.
- `BEANCS_PRIVATE_INGRESS_NAMESPACES`: namespaces that may route Tailnet-only traffic to BeanCS-managed pods, defaults to `tailscale,tailscale-system`.
- `BEANCS_CERT_MANAGER_ISSUER_NAME`: project namespace Issuer name for public app certificates, defaults to `beancs-letsencrypt`.
- `BEANCS_CERT_MANAGER_ACME_SERVER`: ACME directory URL, defaults to Let's Encrypt production.
- `BEANCS_CERT_MANAGER_EMAIL`: optional ACME account email.
- `BEANCS_CERT_MANAGER_CLOUDFLARE_SECRET_NAME`: per-project Cloudflare DNS-01 token Secret name, defaults to `beancs-cloudflare-dns01`.
- `BEANCS_CERT_MANAGER_CLOUDFLARE_SECRET_KEY`: Cloudflare token Secret key, defaults to `api-token`.
- `BEANCS_CERT_MANAGER_PRIVATE_KEY_SECRET_SUFFIX`: ACME account private key Secret suffix, defaults to `account-key`.

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
