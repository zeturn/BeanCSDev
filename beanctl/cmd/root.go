package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zeturn/beanctl/internal/api"
	"github.com/zeturn/beanctl/internal/config"
)

var (
	cfgFile      string
	profileName  string
	apiURLFlag   string
	outputFormat string
	quiet        bool
	verbose      bool
	cfg          *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "beanctl",
	Short: "BeanCS Controller command line client",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cfg != nil {
			return nil
		}
		viper.SetEnvPrefix("BEANCTL")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		viper.AutomaticEnv()
		_ = viper.BindPFlag("api-url", cmd.Root().PersistentFlags().Lookup("api-url"))
		_ = viper.BindPFlag("profile", cmd.Root().PersistentFlags().Lookup("profile"))
		_ = viper.BindPFlag("output", cmd.Root().PersistentFlags().Lookup("output"))
		loaded, err := config.Load(cfgFile)
		if err != nil {
			return err
		}
		cfg = loaded
		if profileName == "" {
			profileName = viper.GetString("profile")
		}
		if profileName == "" {
			profileName = cfg.CurrentProfile
		}
		if outputFormat == "" {
			outputFormat = viper.GetString("output")
		}
		if outputFormat == "" {
			outputFormat = "table"
		}
		if apiURLFlag == "" {
			apiURLFlag = viper.GetString("api-url")
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.beanctl/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURLFlag, "api-url", "", "BeanCS Controller URL")
	rootCmd.PersistentFlags().StringVar(&profileName, "profile", "default", "profile name")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format: table, json, yaml")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "only print core data")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print request debug information")

	rootCmd.AddCommand(loginCmd(), logoutCmd(), whoamiCmd())
	rootCmd.AddCommand(credentialCmd(), projectCmd(), deployCmd(), adminCmd(), completionCmd())
}

func activeProfile() (config.Profile, error) {
	p, ok := cfg.Profile(profileName)
	if !ok {
		return config.Profile{}, fmt.Errorf("profile %q not found", profileName)
	}
	if apiURLFlag != "" {
		p.APIURL = apiURLFlag
	}
	if p.APIURL == "" {
		return config.Profile{}, fmt.Errorf("api-url is required")
	}
	return p, nil
}

func client() (*api.Client, error) {
	p, err := activeProfile()
	if err != nil {
		return nil, err
	}
	return api.New(p.APIURL, profileName, verbose)
}

func ctx() context.Context {
	return context.Background()
}
