# Security Policy

## Supported Versions

Security fixes are expected to target the default branch and the active deployment branch. If long-lived release branches are introduced later, document their support window here.

## Reporting a Vulnerability

Please do not report suspected vulnerabilities in public issues.

Use GitHub's private vulnerability reporting flow when it is available for this repository. If that is not available, contact the repository owner through a private channel and include enough detail to reproduce and assess the issue.

A useful report should include:

- Affected component, such as controller API, `beanctl`, web UI, deployment workflow, or Kubernetes manifests.
- Impact and affected data or privileges.
- Reproduction steps or a minimal proof of concept.
- Relevant versions, commits, configuration, and environment details.
- Any known mitigations.

## Secret Handling

Never commit real values for kubeconfigs, registry tokens, OAuth client secrets, GitHub App private keys, database URLs with credentials, webhook secrets, or encryption keys.

Runtime secrets should be stored in GitHub Secrets, Kubernetes Secrets, or a local secret manager. Development-only examples should use placeholder values and should not point at production services.

## Security Baseline

The project should keep the following checks healthy:

- Go tests and `go vet` for both Go modules.
- Web UI dependency installation and production build.
- CodeQL analysis for Go, JavaScript/TypeScript, and GitHub Actions workflows.
- `govulncheck` advisory scans for Go modules.
- Dependabot updates for Go modules, npm, GitHub Actions, and Docker base images.
- Container image scanning before production deployment when registry tooling is available.
