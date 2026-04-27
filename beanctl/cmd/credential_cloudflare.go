package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/api"
	"github.com/zeturn/beanctl/internal/output"
)

func credentialCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "credential", Aliases: []string{"cred"}, Short: "Manage credentials"}
	cmd.AddCommand(cfCredentialCmd(), ghCredentialCmd(), bpCredentialCmd())
	return cmd
}

func cfCredentialCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cf", Short: "Manage Cloudflare credentials"}
	cmd.AddCommand(cfAddCmd(), listCredCmd("list", "cloudflare"), verifyCredCmd("verify", "cloudflare"), shareCredCmd("share", "cloudflare"), deleteCredCmd("delete", "cloudflare"))
	return cmd
}

func cfAddCmd() *cobra.Command {
	var name, token, zoneID, domain, accountID string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add Cloudflare credential",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if token == "" {
				token, err = promptSecret("Cloudflare API token")
				if err != nil {
					return err
				}
			}
			out, err := c.AddCloudflare(ctx(), map[string]any{"name": name, "api_token": token, "zone_id": zoneID, "domain": domain, "account_id": accountID})
			if err != nil {
				return err
			}
			return printCredential(out)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&token, "token", "", "Cloudflare API token")
	cmd.Flags().StringVar(&zoneID, "zone-id", "", "Cloudflare zone ID")
	cmd.Flags().StringVar(&domain, "domain", "", "domain, e.g. hollodata.com")
	cmd.Flags().StringVar(&accountID, "account-id", "", "Cloudflare account ID")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("zone-id")
	_ = cmd.MarkFlagRequired("domain")
	return cmd
}

func listCredCmd(use, typ string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: "List credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			var list []api.Credential
			switch typ {
			case "cloudflare":
				list, err = c.ListCloudflare(ctx())
			case "github":
				list, err = c.ListGitHub(ctx())
			case "basaltpass":
				list, err = c.ListBasaltPass(ctx())
			}
			if err != nil {
				return err
			}
			return printCredentials(list)
		},
	}
}

func verifyCredCmd(use, typ string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " ID",
		Short: "Verify credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			switch typ {
			case "cloudflare":
				_, err = c.VerifyCloudflare(ctx(), args[0])
			case "github":
				_, err = c.VerifyGitHub(ctx(), args[0])
			case "basaltpass":
				_, err = c.HealthBasaltPass(ctx(), args[0])
			}
			if err != nil {
				return err
			}
			if !quiet {
				fmt.Println("ok")
			}
			return nil
		},
	}
}

func shareCredCmd(use, typ string) *cobra.Command {
	var user, role string
	cmd := &cobra.Command{
		Use:   use + " ID --user USER_ID",
		Short: "Share credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if err := c.ShareCredential(ctx(), typ, args[0], user, role); err != nil {
				return err
			}
			if !quiet {
				fmt.Println("shared")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&user, "user", "", "BasaltPass user ID")
	cmd.Flags().StringVar(&role, "role", "user", "role: owner or user")
	_ = cmd.MarkFlagRequired("user")
	return cmd
}

func deleteCredCmd(use, typ string) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   use + " ID",
		Short: "Delete credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if err := confirm("delete credential " + args[0]); err != nil {
					return err
				}
			}
			c, err := client()
			if err != nil {
				return err
			}
			if err := c.DeleteCredential(ctx(), typ, args[0]); err != nil {
				return err
			}
			if !quiet {
				fmt.Println("deleted")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

func printCredential(cred *api.Credential) error {
	if outputFormat != "table" {
		return output.Print(outputFormat, cred)
	}
	return printCredentials([]api.Credential{*cred})
}

func printCredentials(creds []api.Credential) error {
	if outputFormat != "table" {
		return output.Print(outputFormat, creds)
	}
	rows := [][]string{}
	for _, c := range creds {
		target := c.Domain
		if target == "" {
			target = c.GitOpsRepo
		}
		if target == "" {
			target = c.BaseURL
		}
		rows = append(rows, []string{fmt.Sprint(c.ID), c.Name, target, fmt.Sprint(c.IsActive)})
	}
	return output.Table([]string{"ID", "NAME", "TARGET", "ACTIVE"}, rows)
}

func promptSecret(label string) (string, error) {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return value, nil
}
