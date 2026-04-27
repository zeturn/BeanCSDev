package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/auth"
	"github.com/zeturn/beanctl/internal/config"
)

func loginCmd() *cobra.Command {
	var authURL, clientID, clientSecret string
	var port int
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with BasaltPass using OAuth2 PKCE",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, ok := cfg.Profile(profileName)
			if !ok {
				p = config.Profile{}
			}
			if apiURLFlag != "" {
				p.APIURL = apiURLFlag
			}
			if authURL != "" {
				p.AuthURL = authURL
			}
			if clientID != "" {
				p.ClientID = clientID
			}
			if clientSecret != "" {
				p.ClientSecret = clientSecret
			}
			if p.AuthURL == "" || p.ClientID == "" {
				return fmt.Errorf("auth-url and client-id are required")
			}
			token, err := auth.Login(cmd.Context(), auth.LoginOptions{
				AuthURL:      p.AuthURL,
				ClientID:     p.ClientID,
				ClientSecret: p.ClientSecret,
				Profile:      profileName,
				Port:         port,
			})
			if err != nil {
				return err
			}
			if err := auth.SaveToken(profileName, token); err != nil {
				return err
			}
			cfg.SetProfile(profileName, p)
			cfg.CurrentProfile = profileName
			if err := config.Save(cfgFile, cfg); err != nil {
				return err
			}
			if !quiet {
				fmt.Printf("Login successful for profile %q\n", profileName)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&authURL, "auth-url", "", "BasaltPass auth URL")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret, if required by the BasaltPass instance")
	cmd.Flags().IntVar(&port, "port", 9876, "local callback port")
	return cmd
}
