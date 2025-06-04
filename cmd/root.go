package cmd

import (
	"github.com/spf13/cobra"
)

var (
	tsdbPath   string
	configFile string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "gfs-to-prometheus",
	Short: "Convert GemFire statistics files to Prometheus TSDB",
	Long: `A tool to parse GemFire/Geode statistics files (.gfs) and write
the metrics directly to a Prometheus TSDB for historical analysis.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&tsdbPath, "tsdb-path", "./data", "Path to Prometheus TSDB directory")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file for metric mappings (optional)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}