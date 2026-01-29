package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/n9e/n9e-mcp-server/internal"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "n9e-mcp-server",
	Short: "Nightingale MCP Server",
	Long:  "MCP (Model Context Protocol) server for Nightingale",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to run in stdio mode
		return runStdio(cmd, args)
	},
}

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Run in stdio mode (default)",
	Long:  "Run the MCP server using stdin/stdout for communication",
	RunE:  runStdio,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("n9e-mcp-server %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	// Environment variable prefix
	viper.SetEnvPrefix("N9E")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Global flags
	rootCmd.PersistentFlags().String("token", "", "Nightingale API token (env: N9E_TOKEN)")
	rootCmd.PersistentFlags().String("base-url", "http://localhost:17000", "Nightingale API base URL (env: N9E_BASE_URL)")
	rootCmd.PersistentFlags().StringSlice("toolsets", toolset.DefaultToolsets, "Enabled toolsets (env: N9E_TOOLSETS)")
	rootCmd.PersistentFlags().Bool("read-only", false, "Read-only mode, disable write operations (env: N9E_READ_ONLY)")
	rootCmd.PersistentFlags().String("log-file", "", "Log file path (default: stderr)")

	// Bind to viper
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))
	viper.BindPFlag("toolsets", rootCmd.PersistentFlags().Lookup("toolsets"))
	viper.BindPFlag("read_only", rootCmd.PersistentFlags().Lookup("read-only"))
	viper.BindPFlag("log_file", rootCmd.PersistentFlags().Lookup("log-file"))

	// Add subcommands
	rootCmd.AddCommand(stdioCmd)
	rootCmd.AddCommand(versionCmd)
}

func runStdio(cmd *cobra.Command, args []string) error {
	token := viper.GetString("token")
	if token == "" {
		return fmt.Errorf("N9E_TOKEN is required. Set it via --token flag or N9E_TOKEN environment variable")
	}

	return internal.RunStdioServer(internal.StdioServerConfig{
		Version:         version,
		Token:           token,
		BaseURL:         viper.GetString("base_url"),
		EnabledToolsets: viper.GetStringSlice("toolsets"),
		ReadOnly:        viper.GetBool("read_only"),
		LogFilePath:     viper.GetString("log_file"),
	})
}
