package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/auth"
)

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove saved token from OS keyring",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Delete(profileName); err != nil {
				return err
			}
			if !quiet {
				fmt.Printf("Logged out profile %q\n", profileName)
			}
			return nil
		},
	}
}
