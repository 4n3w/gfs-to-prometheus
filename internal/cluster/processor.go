package cluster

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/4n3w/gfs-to-prometheus/internal/converter"
)

type Config struct {
	ClusterName     string
	NodePatterns    []string
	ExcludePatterns []string
	Recursive       bool
	Concurrency     int
	Converter       *converter.Converter
}

type NodeInfo struct {
	Name     string // e.g., "server-1", "locator-1"
	Type     string // e.g., "server", "locator", "gateway"
	FilePath string
}

type Processor struct {
	config           Config
	excludeRegexes   []*regexp.Regexp
	nodeExtractors   []*NodeExtractor
}

type NodeExtractor struct {
	Pattern *regexp.Regexp
	Name    string
	Type    string
}

func NewProcessor(config Config) (*Processor, error) {
	p := &Processor{
		config: config,
	}

	// Compile exclude patterns
	for _, pattern := range config.ExcludePatterns {
		regex, err := regexp.Compile(globToRegex(pattern))
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %s: %w", pattern, err)
		}
		p.excludeRegexes = append(p.excludeRegexes, regex)
	}

	// Create node extractors for common naming patterns
	p.nodeExtractors = []*NodeExtractor{
		// Docker Compose / Kubernetes patterns
		{
			Pattern: regexp.MustCompile(`([^/]+)/(stats|data|logs)/([^/]*-stats\.gfs)`),
			Name:    "$1",     // Use directory name as node name
			Type:    "server", // Default type, will be refined below
		},
		{
			Pattern: regexp.MustCompile(`.*?([a-zA-Z]+-\d+)[^/]*-stats\.gfs`),
			Name:    "$1",     // Extract node-1, server-2, etc.
			Type:    "server",
		},
		// Traditional patterns
		{
			Pattern: regexp.MustCompile(`.*/([^/]+)/stats/.*\.gfs`),
			Name:    "$1",
			Type:    "server",
		},
		{
			Pattern: regexp.MustCompile(`.*?([^/]+)-stats\.gfs`),
			Name:    "$1",
			Type:    "server",
		},
	}

	return p, nil
}

func (p *Processor) ProcessDirectory(rootDir string) error {
	files, err := p.discoverFiles(rootDir)
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	if len(files) == 0 {
		log.Printf("No GFS files found in %s", rootDir)
		return nil
	}

	log.Printf("Found %d GFS files to process", len(files))

	// Process files with concurrency control
	semaphore := make(chan struct{}, p.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, nodeInfo := range files {
		wg.Add(1)
		go func(node NodeInfo) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			if err := p.processFile(node); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to process %s: %w", node.FilePath, err))
				mu.Unlock()
			}
		}(nodeInfo)
	}

	wg.Wait()

	if len(errors) > 0 {
		log.Printf("Encountered %d errors during processing:", len(errors))
		for _, err := range errors {
			log.Printf("  %v", err)
		}
		return fmt.Errorf("processing completed with %d errors", len(errors))
	}

	return nil
}

func (p *Processor) discoverFiles(rootDir string) ([]NodeInfo, error) {
	var files []NodeInfo
	
	for _, pattern := range p.config.NodePatterns {
		// Convert pattern to absolute path
		searchPattern := filepath.Join(rootDir, pattern)
		
		matches, err := filepath.Glob(searchPattern)
		if err != nil {
			log.Printf("Warning: invalid pattern %s: %v", pattern, err)
			continue
		}

		for _, match := range matches {
			// Check if file should be excluded
			if p.shouldExclude(match) {
				continue
			}

			// Extract node information
			nodeInfo := p.extractNodeInfo(match)
			if nodeInfo.Name != "" {
				files = append(files, nodeInfo)
				log.Printf("Discovered: %s (node=%s, type=%s)", match, nodeInfo.Name, nodeInfo.Type)
			}
		}
	}

	return files, nil
}

func (p *Processor) shouldExclude(path string) bool {
	for _, regex := range p.excludeRegexes {
		if regex.MatchString(path) {
			return true
		}
	}
	return false
}

func (p *Processor) extractNodeInfo(filePath string) NodeInfo {
	nodeInfo := NodeInfo{
		FilePath: filePath,
		Name:     "unknown",
		Type:     "server", // Default
	}

	// Try each extractor pattern
	for _, extractor := range p.nodeExtractors {
		if matches := extractor.Pattern.FindStringSubmatch(filePath); matches != nil {
			// Replace placeholders in name and type
			name := extractor.Name
			nodeType := extractor.Type
			
			for i, match := range matches {
				placeholder := fmt.Sprintf("$%d", i)
				name = strings.ReplaceAll(name, placeholder, match)
				nodeType = strings.ReplaceAll(nodeType, placeholder, match)
			}
			
			nodeInfo.Name = name
			nodeInfo.Type = p.inferNodeType(name, filePath)
			break
		}
	}

	return nodeInfo
}

func (p *Processor) inferNodeType(nodeName, filePath string) string {
	nameLower := strings.ToLower(nodeName)
	pathLower := strings.ToLower(filePath)
	
	// Check for common node type indicators
	if strings.Contains(nameLower, "locator") || strings.Contains(pathLower, "locator") {
		return "locator"
	}
	if strings.Contains(nameLower, "gateway") || strings.Contains(pathLower, "gateway") {
		return "gateway"
	}
	if strings.Contains(nameLower, "server") || strings.Contains(pathLower, "server") {
		return "server"
	}
	
	// Default to server
	return "server"
}

func (p *Processor) processFile(nodeInfo NodeInfo) error {
	log.Printf("Processing %s (cluster=%s, node=%s, type=%s)", 
		nodeInfo.FilePath, p.config.ClusterName, nodeInfo.Name, nodeInfo.Type)

	// Set cluster labels for this file
	originalConverter := p.config.Converter
	clusterConverter := &ClusterConverter{
		Converter:   originalConverter,
		ClusterName: p.config.ClusterName,
		NodeName:    nodeInfo.Name,
		NodeType:    nodeInfo.Type,
	}

	// Process the file with cluster-aware converter
	return clusterConverter.ConvertFile(nodeInfo.FilePath)
}

// Convert glob pattern to regex
func globToRegex(glob string) string {
	// Escape regex special characters except * and ?
	regex := regexp.QuoteMeta(glob)
	
	// Convert glob wildcards to regex
	regex = strings.ReplaceAll(regex, `\*`, `.*`)
	regex = strings.ReplaceAll(regex, `\?`, `.`)
	
	// Anchor the pattern
	return "^" + regex + "$"
}