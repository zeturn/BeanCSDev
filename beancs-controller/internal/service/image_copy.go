package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type registryAuth struct {
	Username string
	Password string
}

func copyContainerImage(ctx context.Context, sourceRef, targetRef string, sourceAuth, targetAuth registryAuth) error {
	sourceRef = strings.TrimSpace(sourceRef)
	targetRef = strings.TrimSpace(targetRef)
	if sourceRef == "" || targetRef == "" {
		return fmt.Errorf("source and target image references are required")
	}
	src, err := name.ParseReference(sourceRef, name.WeakValidation)
	if err != nil {
		return fmt.Errorf("source image reference: %w", err)
	}
	dst, err := name.ParseReference(targetRef, name.WeakValidation)
	if err != nil {
		return fmt.Errorf("target image reference: %w", err)
	}
	img, err := remote.Image(src, remote.WithContext(ctx), remote.WithAuth(registryAuthenticator(sourceAuth)))
	if err != nil {
		return fmt.Errorf("pull source image %s: %w", sourceRef, err)
	}
	if err := remote.Write(dst, img, remote.WithContext(ctx), remote.WithAuth(registryAuthenticator(targetAuth))); err != nil {
		return fmt.Errorf("push target image %s: %w", targetRef, err)
	}
	return nil
}

func registryAuthenticator(auth registryAuth) authn.Authenticator {
	if strings.TrimSpace(auth.Username) == "" && strings.TrimSpace(auth.Password) == "" {
		return authn.Anonymous
	}
	return &authn.Basic{
		Username: strings.TrimSpace(auth.Username),
		Password: auth.Password,
	}
}
