package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/4n3w/gfs-to-prometheus/internal/cluster"
	"github.com/4n3w/gfs-to-prometheus/internal/converter"
	"github.com/spf13/cobra"
)

var (
	clusterName    string
	nodePatterns   []string
	excludePatterns []string
	recursive      bool
	concurrency    int
)

var clusterCmd = &cobra.Command{
	Use:   "cluster [directories...]",
	Short: "Process GFS files from entire GemFire cluster",
	Long: `Process GFS statistics files from multiple nodes in a GemFire cluster.
Supports flexible file discovery for various deployment patterns including
Docker Compose, Kubernetes, and traditional deployments.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conv, err := converter.New(tsdbPath, configFile)
		if err != nil {
			return fmt.Errorf("failed to initialize converter: %w", err)
		}
		defer conv.Close()

		processor, err := cluster.NewProcessor(cluster.Config{
			ClusterName:     clusterName,
			NodePatterns:    nodePatterns,
			ExcludePatterns: excludePatterns,
			Recursive:       recursive,
			Concurrency:     concurrency,
			Converter:       conv,
		})
		if err != nil {
			return fmt.Errorf("failed to create cluster processor: %w", err)
		}

		for _, dir := range args {
			fmt.Printf("Processing cluster directory: %s\n", dir)
			if err := processor.ProcessDirectory(dir); err != nil {
				return fmt.Errorf("failed to process directory %s: %w", dir, err)
			}
		}

		fmt.Println("Cluster processing complete!")
		return nil
	},
}

var clusterWatchCmd = &cobra.Command{
	Use:   "cluster-watch [directories...]",
	Short: "Watch directories for new GFS files from cluster nodes",
	Long: `Continuously monitor directories for new or modified GFS files from
multiple cluster nodes. Supports the same flexible patterns as cluster command.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conv, err := converter.New(tsdbPath, configFile)
		if err != nil {
			return fmt.Errorf("failed to initialize converter: %w", err)
		}
		defer conv.Close()

		processor, err := cluster.NewProcessor(cluster.Config{
			ClusterName:     clusterName,
			NodePatterns:    nodePatterns,
			ExcludePatterns: excludePatterns,
			Recursive:       recursive,
			Concurrency:     concurrency,
			Converter:       conv,
		})
		if err != nil {
			return fmt.Errorf("failed to create cluster processor: %w", err)
		}

		watcher, err := cluster.NewWatcher(processor)
		if err != nil {
			return fmt.Errorf("failed to create cluster watcher: %w", err)
		}
		defer watcher.Close()

		for _, dir := range args {
			absDir, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("invalid directory %s: %w", dir, err)
			}
			if err := watcher.AddDirectory(absDir); err != nil {
				return fmt.Errorf("failed to watch %s: %w", absDir, err)
			}
			log.Printf("Watching cluster directory: %s", absDir)
		}

		fmt.Println("Watching for cluster GFS files... Press Ctrl+C to stop.")
		return watcher.Start()
	},
}

func init() {
	// Common flags for both cluster commands
	for _, cmd := range []*cobra.Command{clusterCmd, clusterWatchCmd} {
		cmd.Flags().StringVar(&clusterName, "cluster-name", "gemfire", "Name of the cluster for labeling")
		cmd.Flags().StringSliceVar(&nodePatterns, "node-pattern", []string{
			// Docker Compose patterns
			"*/stats/*-stats.gfs",           // compose/server-1/stats/server-1-stats.gfs
			"*/*/*-stats.gfs",               // volumes/server-1/data/server-1-stats.gfs
			"*/data/*-stats.gfs",            // server-1/data/server-1-stats.gfs
			
			// Traditional patterns  
			"*/stats/*.gfs",                 // server-1/stats/statistics.gfs
			"*/*-stats.gfs",                 // server-1/server-1-stats.gfs
			
			// Kubernetes patterns
			"*/persistent-data/*-stats.gfs", // server-1/persistent-data/server-1-stats.gfs
			"*/logs/*-stats.gfs",            // server-1/logs/server-1-stats.gfs
		}, "Patterns for finding node stats files (supports glob)")
		
		cmd.Flags().StringSliceVar(&excludePatterns, "exclude", []string{
			"*/tmp/*",
			"*/temp/*", 
			"*/.git/*",
			"*/node_modules/*",
		}, "Patterns to exclude from search")
		
		cmd.Flags().BoolVar(&recursive, "recursive", true, "Search directories recursively")
		cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Number of files to process concurrently")
	}

	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(clusterWatchCmd)
}