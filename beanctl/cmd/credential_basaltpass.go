package cmd

import "github.com/spf13/cobra"

func bpCredentialCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "bp", Short: "Manage BasaltPass app instances"}
	cmd.AddCommand(bpAddCmd(), listCredCmd("list", "basaltpass"), verifyCredCmd("health", "basaltpass"), shareCredCmd("share", "basaltpass"), deleteCredCmd("delete", "basaltpass"))
	return cmd
}

func bpAddCmd() *cobra.Command {
	var name, url, clientID, clientSecret, serviceToken string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add BasaltPass app instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if clientSecret == "" {
				clientSecret, err = promptSecret("BasaltPass client secret")
				if err != nil {
					return err
				}
			}
			body := map[string]any{"name": name, "base_url": url, "client_id": clientID, "client_secret": clientSecret}
			if serviceToken != "" {
				body["service_token"] = serviceToken
			}
			out, err := c.AddBasaltPass(ctx(), body)
			if err != nil {
				return err
			}
			return printCredential(out)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&url, "url", "", "BasaltPass base URL")
	cmd.Flags().StringVar(&clientID, "client-id", "", "BeanCS client ID in this instance")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "BeanCS client secret in this instance")
	cmd.Flags().StringVar(&serviceToken, "service-token", "", "tenant/admin bearer token for app registration when client_credentials is unavailable")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("client-id")
	return cmd
}
