# Contributing

Thanks for helping improve BeanCSDev. This guide describes the checks expected before opening or merging changes.

## Code of Conduct

All participation in this project is covered by `CODE_OF_CONDUCT.md`. Please keep discussions respectful, constructive, and focused on improving the project.

## Development Setup

1. Install Go matching the versions declared in the module files.
2. Install Node.js 24 and npm for the web UI.
3. Install Docker if you need to build the controller image locally.
4. Configure local environment variables for the controller. Never commit real secrets or kubeconfigs.

## Local Checks

Run controller checks:

```bash
cd beancs-controller
go test ./...
go vet ./...
go build ./...
```

Run CLI checks:

```bash
cd beanctl
go test ./...
go vet ./...
go build ./...
```

Run web checks:

```bash
cd beancs-controller/internal/web
npm ci
npm run build
```

Build the Docker image when touching the Dockerfile, web build, or controller packaging:

```bash
cd beancs-controller
docker build -t beancs-controller:dev .
```

## Pull Request Checklist

- Keep changes focused and explain the user-visible behavior change.
- Add or update tests for service, middleware, configuration, and CLI behavior where practical.
- Update `README.md` or deployment docs when setup, configuration, or workflows change.
- Run the relevant local checks before requesting review.
- Do not include generated credentials, local kubeconfigs, private keys, tokens, or `.env` files.
- For deployment changes, describe rollback considerations and required secret or variable changes.

## Branching and Release Notes

The default branch is `master`. The `deploy` branch and version tags are used by the k3s deployment workflow. Release tags should be created from commits that are already contained in `origin/deploy`.

## License

By contributing to this repository, you agree that your contributions are licensed under the ISC License unless a different written agreement applies.
