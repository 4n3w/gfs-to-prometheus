package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/4n3w/gfs-to-prometheus/internal/converter"
	"github.com/4n3w/gfs-to-prometheus/internal/watcher"
	"github.com/spf13/cobra"
)

var (
	watchDirs []string
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch directories for new GFS files",
	Long:  `Continuously monitor directories for new or modified GFS files and convert them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		conv, err := converter.New(tsdbPath, configFile)
		if err != nil {
			return fmt.Errorf("failed to initialize converter: %w", err)
		}
		defer conv.Close()

		w, err := watcher.New(conv)
		if err != nil {
			return fmt.Errorf("failed to create watcher: %w", err)
		}
		defer w.Close()

		for _, dir := range watchDirs {
			absDir, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("invalid directory %s: %w", dir, err)
			}
			if err := w.AddDirectory(absDir); err != nil {
				return fmt.Errorf("failed to watch %s: %w", absDir, err)
			}
			log.Printf("Watching directory: %s", absDir)
		}

		fmt.Println("Watching for GFS files... Press Ctrl+C to stop.")
		return w.Start()
	},
}

func init() {
	watchCmd.Flags().StringSliceVar(&watchDirs, "dir", []string{"."}, "Directories to watch for GFS files")
	rootCmd.AddCommand(watchCmd)
}