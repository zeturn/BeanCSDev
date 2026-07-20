# BeanCSDev

BeanCSDev is the development repository for BeanCS, a control panel for California Beans Cloud Service. The repository contains the BeanCS controller service, the `beanctl` command-line tool, web UI assets, Kubernetes deployment manifests, and GitHub Actions workflows for CI/CD to k3s.

## Repository Layout

- `beancs-controller/` - Go controller service, embedded web UI, Kubernetes integration, Dockerfile, and deployment manifests.
- `beancs-controller/cmd/server/` - controller entrypoint.
- `beancs-controller/internal/` - application packages for config, handlers, middleware, services, models, migrations, Kubernetes operations, registry operations, and web assets.
- `beancs-controller/internal/web/` - Vite/React/Tailwind web UI.
- `beanctl/` - Go CLI for interacting with BeanCS.
- `.github/workflows/` - CI/CD, CodeQL, k3s diagnostics, and maintenance workflows.
- `GITHUB_ACTIONS_K3S_DEPLOY.md` - production deployment workflow notes.

## Prerequisites

- Go matching the module versions in `beancs-controller/go.mod` and `beanctl/go.mod`.
- Node.js 24 and npm for the web UI.
- Docker for local image builds.
- Kubernetes/k3s access only when deploying or running cluster integration workflows.
- PostgreSQL or a compatible `DATABASE_URL` for running the controller locally.

## Local Development

Build and test the controller:

```bash
cd beancs-controller
go test ./...
go vet ./...
go build ./...
```

Build the web UI:

```bash
cd beancs-controller/internal/web
npm ci
npm run build
```

Run the controller locally after setting the required environment variables:

```bash
cd beancs-controller
go run ./cmd/server
```

Run the CLI:

```bash
cd beanctl
go run . --help
go test ./...
go vet ./...
```

## Controller Configuration

The controller is configured through environment variables. The most important required values are:

- `DATABASE_URL` - PostgreSQL connection string.
- `BP_MGMT_BASE_URL` - BasaltPass management API base URL.
- `BP_MGMT_CLIENT_ID` - BasaltPass management OAuth client ID.
- `BP_MGMT_CLIENT_SECRET` - BasaltPass management OAuth client secret.
- `WEBHOOK_SECRET` - shared secret for webhook verification.
- `ENCRYPTION_KEY` - 64 hex characters used for AES-256-GCM encryption.

Common optional values include:

- `PORT` - HTTP port, default `8080`.
- `VERSION` - version string returned by health/version endpoints.
- `CORS_ORIGINS` - allowed CORS origins, default `*`.
- `BEANCS_API_RATE_LIMIT_PER_MINUTE` - authenticated API rate limit, default `600`.
- `BEANCS_WEBHOOK_RATE_LIMIT_PER_MINUTE` - webhook rate limit, default `300`.
- `BEANCS_PUBLIC_HOST`, `BEANCS_WEBHOOK_HOST`, `BEANCS_TAILSCALE_HOST` - public and private route hostnames.
- `BEANCS_REGISTRY_HOST`, `BEANCS_REGISTRY_USERNAME`, `BEANCS_REGISTRY_TOKEN` - registry settings.

See `beancs-controller/internal/config/config.go` for the full configuration surface.

## Docker

Build the controller image locally:

```bash
cd beancs-controller
docker build -t beancs-controller:dev .
```

The Dockerfile builds the web UI first, compiles the Go controller, and packages the binary into an Alpine runtime image.

## CI/CD and Security Scanning

This repository has several workflow categories:

- `CI` runs lightweight validation on pull requests and pushes to `master`, `main`, and `deploy`.
- `CodeQL` scans Go, JavaScript/TypeScript, and GitHub Actions workflow code.
- `Harbor + k3s CI/CD` builds and publishes the controller image, then deploys tagged releases to k3s.

Deployment setup and required secrets are documented in `GITHUB_ACTIONS_K3S_DEPLOY.md`.

## Security

Do not commit kubeconfigs, tokens, passwords, private keys, or generated `.env` files. Runtime secrets should be provided through GitHub Secrets, Kubernetes Secrets, or a local secret manager.

Please see `SECURITY.md` before reporting a vulnerability.

## Contributing

Please see `CONTRIBUTING.md` for the local checks and pull request expectations. All project participation is covered by `CODE_OF_CONDUCT.md`.

## License

This project is licensed under the ISC License. See `LICENSE` for details.
