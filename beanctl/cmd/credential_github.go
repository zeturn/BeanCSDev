package cmd

import (
	"github.com/spf13/cobra"
)

func ghCredentialCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "gh", Short: "Manage GitHub credentials"}
	cmd.AddCommand(ghAddCmd(), listCredCmd("list", "github"), verifyCredCmd("verify", "github"), shareCredCmd("share", "github"), deleteCredCmd("delete", "github"))
	return cmd
}

func ghAddCmd() *cobra.Command {
	var name, token, org, repo string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add GitHub credential",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if token == "" {
				token, err = promptSecret("GitHub token")
				if err != nil {
					return err
				}
			}
			out, err := c.AddGitHub(ctx(), map[string]any{"name": name, "token": token, "org": org, "gitops_repo": repo})
			if err != nil {
				return err
			}
			return printCredential(out)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&token, "token", "", "GitHub token")
	cmd.Flags().StringVar(&org, "org", "", "GitHub org")
	cmd.Flags().StringVar(&repo, "gitops-repo", "", "GitOps repo")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("gitops-repo")
	return cmd
}
