package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/api"
	"github.com/zeturn/beanctl/internal/output"
)

func deployCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "deploy", Short: "Manage deployments"}
	cmd.AddCommand(deployListCmd(), deployGetCmd(), deployRollbackCmd())
	return cmd
}

func deployListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list PROJECT",
		Short: "List deployments",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			list, err := c.ListDeployments(ctx(), args[0])
			if err != nil {
				return err
			}
			return printDeployments(list)
		},
	}
}

func deployGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get PROJECT DEPLOYMENT_ID",
		Short: "Get deployment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			dep, err := c.GetDeployment(ctx(), args[0], args[1])
			if err != nil {
				return err
			}
			if outputFormat != "table" {
				return output.Print(outputFormat, dep)
			}
			return printDeployments([]api.Deployment{*dep})
		},
	}
}

func deployRollbackCmd() *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "rollback PROJECT --to DEPLOYMENT_ID",
		Short: "Rollback deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if err := c.Rollback(ctx(), args[0], to); err != nil {
				return err
			}
			if !quiet {
				fmt.Println("rollback triggered")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "deployment ID")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func printDeployments(deps []api.Deployment) error {
	if outputFormat != "table" {
		return output.Print(outputFormat, deps)
	}
	rows := [][]string{}
	for _, d := range deps {
		rows = append(rows, []string{fmt.Sprint(d.ID), d.Tag, d.Status, short(d.ImageDigest), short(d.CommitSHA), d.TriggeredBy, d.CreatedAt.Local().String()})
	}
	return output.Table([]string{"ID", "TAG", "STATUS", "DIGEST", "COMMIT", "TRIGGERED_BY", "TIME"}, rows)
}

func short(v string) string {
	if len(v) > 12 {
		return v[:12]
	}
	return v
}
