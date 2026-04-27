package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/output"
)

func projectStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status PROJECT",
		Short: "Show pod status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			out, err := c.Status(ctx(), args[0])
			if err != nil {
				return err
			}
			return output.Print("json", out)
		},
	}
}

func projectLogsCmd() *cobra.Command {
	var tail int
	cmd := &cobra.Command{
		Use:   "logs PROJECT",
		Short: "Show logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			out, err := c.Logs(ctx(), args[0], tail)
			if err != nil {
				return err
			}
			if outputFormat != "table" {
				return output.Print(outputFormat, out)
			}
			fmt.Print(out["logs"])
			return nil
		},
	}
	cmd.Flags().IntVar(&tail, "tail", 100, "number of lines")
	cmd.Flags().Bool("follow", false, "reserved for streaming logs")
	return cmd
}

func projectRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart PROJECT",
		Short: "Rolling restart",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if err := c.Restart(ctx(), args[0]); err != nil {
				return err
			}
			if !quiet {
				fmt.Println("restart triggered")
			}
			return nil
		},
	}
}

func projectScaleCmd() *cobra.Command {
	var replicas int
	cmd := &cobra.Command{
		Use:   "scale PROJECT --replicas N",
		Short: "Scale deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			if err := c.Scale(ctx(), args[0], replicas); err != nil {
				return err
			}
			if !quiet {
				fmt.Println("scaled")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&replicas, "replicas", 1, "replicas")
	return cmd
}
