package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/4n3w/gfs-to-prometheus/internal/converter"
	"github.com/spf13/cobra"
)

var convertCmd = &cobra.Command{
	Use:   "convert [gfs files...]",
	Short: "Convert GFS files to Prometheus TSDB",
	Long:  `Process one or more GFS files and write their metrics to Prometheus TSDB.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conv, err := converter.New(tsdbPath, configFile)
		if err != nil {
			return fmt.Errorf("failed to initialize converter: %w", err)
		}
		defer conv.Close()

		for _, pattern := range args {
			files, err := filepath.Glob(pattern)
			if err != nil {
				return fmt.Errorf("invalid file pattern %s: %w", pattern, err)
			}

			for _, file := range files {
				fmt.Printf("Processing %s...\n", file)
				if err := conv.ConvertFile(file); err != nil {
					return fmt.Errorf("failed to convert %s: %w", file, err)
				}
			}
		}

		fmt.Println("Conversion complete!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)
}