package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/output"
)

func adminCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "admin", Short: "Admin commands"}
	cmd.AddCommand(adminOverviewCmd(), adminNodesCmd(), adminQuotasCmd())
	return cmd
}

func adminOverviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "overview",
		Short: "Cluster overview",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			out, err := c.Overview(ctx())
			if err != nil {
				return err
			}
			return output.Print("json", out)
		},
	}
}

func adminNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nodes",
		Short: "List nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			out, err := c.Nodes(ctx())
			if err != nil {
				return err
			}
			return output.Print("json", out)
		},
	}
}

func adminQuotasCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "quotas", Short: "Manage quotas"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List quotas",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			out, err := c.Quotas(ctx())
			if err != nil {
				return err
			}
			return output.Print("json", out)
		},
	}, adminQuotaSetCmd())
	return cmd
}

func adminQuotaSetCmd() *cobra.Command {
	var maxProjects, maxCPU, maxMemory int
	cmd := &cobra.Command{
		Use:   "set TEAM_ID",
		Short: "Set team quota",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("max-projects") {
				body["max_projects"] = maxProjects
			}
			if cmd.Flags().Changed("max-cpu") {
				body["max_cpu_millis"] = maxCPU
			}
			if cmd.Flags().Changed("max-memory") {
				body["max_memory_mb"] = maxMemory
			}
			c, err := client()
			if err != nil {
				return err
			}
			out, err := c.SetQuota(ctx(), args[0], body)
			if err != nil {
				return err
			}
			if outputFormat != "table" {
				return output.Print(outputFormat, out)
			}
			fmt.Println("quota updated")
			return nil
		},
	}
	cmd.Flags().IntVar(&maxProjects, "max-projects", 0, "max projects")
	cmd.Flags().IntVar(&maxCPU, "max-cpu", 0, "max CPU millis")
	cmd.Flags().IntVar(&maxMemory, "max-memory", 0, "max memory MB")
	return cmd
}
