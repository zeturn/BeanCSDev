package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/output"
)

func projectEnvCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "env", Short: "Manage project environment variables"}
	cmd.AddCommand(projectEnvListCmd(), projectEnvSetCmd(), projectEnvUnsetCmd())
	return cmd
}

func projectEnvListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list PROJECT",
		Short: "List masked env vars",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			env, err := c.GetEnv(ctx(), args[0])
			if err != nil {
				return err
			}
			if outputFormat != "table" {
				return output.Print(outputFormat, env)
			}
			rows := [][]string{}
			for k, v := range env {
				rows = append(rows, []string{k, v})
			}
			return output.Table([]string{"KEY", "VALUE"}, rows)
		},
	}
}

func projectEnvSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set PROJECT KEY=VALUE [KEY=VALUE...]",
		Short: "Set env vars",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if err := c.PutEnv(ctx(), args[0], parseEnvPairs(args[1:])); err != nil {
				return err
			}
			if !quiet {
				fmt.Println("env updated")
			}
			return nil
		},
	}
}

func projectEnvUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unset PROJECT KEY [KEY...]",
		Short: "Unset env vars",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("unset is not supported by the current Controller API without exposing plaintext values; use env set with the desired full env set")
		},
	}
}
