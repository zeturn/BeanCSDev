package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

type LoginOptions struct {
	AuthURL      string
	ClientID     string
	ClientSecret string
	Profile      string
	Port         int
	Scopes       []string
}

func Login(ctx context.Context, opts LoginOptions) (*oauth2.Token, error) {
	if opts.Port == 0 {
		opts.Port = 9876
	}
	if len(opts.Scopes) == 0 {
		opts.Scopes = []string{"openid", "profile", "email", "beancs.admin"}
	}
	verifier, err := randomVerifier()
	if err != nil {
		return nil, err
	}
	state, err := randomVerifier()
	if err != nil {
		return nil, err
	}
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", opts.Port)
	conf := oauth2.Config{
		ClientID:     opts.ClientID,
		ClientSecret: opts.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       opts.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  trimSlash(opts.AuthURL) + "/oauth/authorize",
			TokenURL: trimSlash(opts.AuthURL) + "/oauth/token",
		},
	}

	codeChal := codeChallenge(verifier)
	authURL := conf.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChal),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", opts.Port)}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid state", http.StatusBadRequest)
			errCh <- fmt.Errorf("invalid OAuth state")
			return
		}
		if e := r.URL.Query().Get("error"); e != "" {
			http.Error(w, e, http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth error: %s", e)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- fmt.Errorf("missing authorization code")
			return
		}
		_, _ = w.Write([]byte("BeanCS login complete. You can close this window."))
		codeCh <- code
	})
	server.Handler = mux

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, err
	}
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := openBrowser(authURL); err != nil {
		fmt.Println("Open this URL in your browser:")
		fmt.Println(authURL)
	}

	select {
	case code := <-codeCh:
		return conf.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func randomVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func trimSlash(v string) string {
	u, err := url.Parse(v)
	if err != nil {
		return v
	}
	u.RawQuery = ""
	u.Fragment = ""
	out := u.String()
	for len(out) > 0 && out[len(out)-1] == '/' {
		out = out[:len(out)-1]
	}
	return out
}

func openBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	case "darwin":
		return exec.Command("open", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}
