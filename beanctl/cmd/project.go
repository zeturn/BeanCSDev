package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeturn/beanctl/internal/api"
	"github.com/zeturn/beanctl/internal/output"
)

func projectCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "project", Short: "Manage projects"}
	cmd.AddCommand(projectCreateCmd(), projectListCmd(), projectGetCmd(), projectUpdateCmd(), projectDeleteCmd())
	cmd.AddCommand(projectEnvCmd(), projectStatusCmd(), projectLogsCmd(), projectRestartCmd(), projectScaleCmd())
	return cmd
}

func projectCreateCmd() *cobra.Command {
	var repo, branch, dockerfile, exposure, subdomain, preset, portsJSON string
	var ghID, bpID, cfID uint
	var port, replicas int
	var envPairs []string
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ports, err := parseProjectPorts(portsJSON)
			if err != nil {
				return err
			}
			body := map[string]any{
				"name":                   args[0],
				"github_repo":            repo,
				"github_branch":          branch,
				"dockerfile_path":        dockerfile,
				"github_credential_id":   ghID,
				"basaltpass_instance_id": bpID,
				"exposure_mode":          exposure,
				"subdomain":              subdomain,
				"resource_preset":        preset,
				"port":                   port,
				"ports":                  ports,
				"replicas":               replicas,
				"env":                    parseEnvPairs(envPairs),
			}
			if cfID != 0 {
				body["cloudflare_credential_id"] = cfID
			}
			c, err := client()
			if err != nil {
				return err
			}
			out, err := c.CreateProject(ctx(), body)
			if err != nil {
				return err
			}
			return printProject(out)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo owner/name")
	cmd.Flags().StringVar(&branch, "branch", "main", "GitHub branch")
	cmd.Flags().StringVar(&dockerfile, "dockerfile", "Dockerfile", "Dockerfile path")
	cmd.Flags().UintVar(&ghID, "gh-credential", 0, "GitHub credential ID")
	cmd.Flags().UintVar(&bpID, "bp-credential", 0, "BasaltPass credential ID")
	cmd.Flags().UintVar(&cfID, "cf-credential", 0, "Cloudflare credential ID")
	cmd.Flags().StringVar(&exposure, "exposure", "private", "public, private, internal-only")
	cmd.Flags().StringVar(&subdomain, "subdomain", "", "public subdomain")
	cmd.Flags().StringVar(&portsJSON, "ports-json", "", `port JSON, for example: [{"name":"web","port":8080,"exposure":"public","domain":"app.example.com"}]`)
	cmd.Flags().StringVar(&preset, "preset", "small", "resource preset")
	cmd.Flags().IntVar(&port, "port", 8080, "container port")
	cmd.Flags().IntVar(&replicas, "replicas", 1, "replicas")
	cmd.Flags().StringArrayVar(&envPairs, "env", nil, "environment variable KEY=VALUE")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("gh-credential")
	_ = cmd.MarkFlagRequired("bp-credential")
	_ = cmd.MarkFlagRequired("ports-json")
	return cmd
}

func projectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			list, err := c.ListProjects(ctx())
			if err != nil {
				return err
			}
			return printProjects(list)
		},
	}
}

func projectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get PROJECT",
		Short: "Get project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client()
			if err != nil {
				return err
			}
			p, err := c.GetProject(ctx(), args[0])
			if err != nil {
				return err
			}
			return printProject(p)
		},
	}
}

func projectUpdateCmd() *cobra.Command {
	var replicas, port int
	var preset, status string
	cmd := &cobra.Command{
		Use:   "update PROJECT",
		Short: "Update project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("replicas") {
				body["replicas"] = replicas
			}
			if cmd.Flags().Changed("port") {
				body["port"] = port
			}
			if preset != "" {
				body["resource_preset"] = preset
			}
			if status != "" {
				body["status"] = status
			}
			c, err := client()
			if err != nil {
				return err
			}
			p, err := c.UpdateProject(ctx(), args[0], body)
			if err != nil {
				return err
			}
			return printProject(p)
		},
	}
	cmd.Flags().IntVar(&replicas, "replicas", 0, "replicas")
	cmd.Flags().IntVar(&port, "port", 0, "port")
	cmd.Flags().StringVar(&preset, "preset", "", "resource preset")
	cmd.Flags().StringVar(&status, "status", "", "status")
	return cmd
}

func projectDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete PROJECT",
		Short: "Delete project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if err := confirm(args[0]); err != nil {
					return err
				}
			}
			c, err := client()
			if err != nil {
				return err
			}
			if err := c.DeleteProject(ctx(), args[0]); err != nil {
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

func printProject(p *api.Project) error {
	if outputFormat != "table" {
		return output.Print(outputFormat, p)
	}
	return printProjects([]api.Project{*p})
}

func printProjects(projects []api.Project) error {
	if outputFormat != "table" {
		return output.Print(outputFormat, projects)
	}
	rows := [][]string{}
	for _, p := range projects {
		domain := p.Domain
		if domain == "" {
			domain = "(" + p.ExposureMode + ")"
		}
		rows = append(rows, []string{fmt.Sprint(p.ID), p.Name, p.Status, domain, summarizePorts(p.Ports), p.ResourcePreset, fmt.Sprint(p.Replicas)})
	}
	return output.Table([]string{"ID", "NAME", "STATUS", "DOMAIN", "PORTS", "PRESET", "REPLICAS"}, rows)
}

func parseEnvPairs(pairs []string) map[string]string {
	out := map[string]string{}
	for _, pair := range pairs {
		k, v, ok := strings.Cut(pair, "=")
		if ok && k != "" {
			out[k] = v
		}
	}
	return out
}

func parseProjectPorts(raw string) ([]map[string]any, error) {
	var ports []map[string]any
	if err := json.Unmarshal([]byte(raw), &ports); err != nil {
		return nil, fmt.Errorf("invalid --ports-json: %w", err)
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("--ports-json must contain at least one port")
	}
	return ports, nil
}

func summarizePorts(ports []api.ProjectPort) string {
	if len(ports) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		parts = append(parts, fmt.Sprintf("%s:%d/%s", p.Name, p.Port, p.Exposure))
	}
	return strings.Join(parts, ",")
}
