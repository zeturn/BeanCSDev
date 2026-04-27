package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/auth"
	"github.com/zeturn/beanctl/internal/output"
)

func whoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current login information",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := auth.LoadToken(profileName)
			if err != nil {
				return err
			}
			claims := auth.Claims(token.AccessToken)
			info := map[string]any{
				"profile":   profileName,
				"user_id":   claims["sub"],
				"tenant_id": claims["tenant_id"],
				"scope":     claims["scope"],
				"expiry":    auth.ExpiryString(token),
			}
			if outputFormat != "table" {
				return output.Print(outputFormat, info)
			}
			return output.Table([]string{"PROFILE", "USER_ID", "TENANT_ID", "SCOPES", "EXPIRY"}, [][]string{{
				profileName,
				fmt.Sprint(info["user_id"]),
				fmt.Sprint(info["tenant_id"]),
				fmt.Sprint(info["scope"]),
				fmt.Sprint(info["expiry"]),
			}})
		},
	}
}
