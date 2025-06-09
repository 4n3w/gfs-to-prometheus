package converter

import (
	"fmt"
	"log"
	"strings"

	"github.com/4n3w/gfs-to-prometheus/internal/config"
	"github.com/4n3w/gfs-to-prometheus/internal/gfs"
	"github.com/4n3w/gfs-to-prometheus/internal/tsdb"
)

type Converter struct {
	writer *tsdb.Writer
	config *config.Config
}

func New(tsdbPath string, configFile string) (*Converter, error) {
	writer, err := tsdb.NewWriter(tsdbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create TSDB writer: %w", err)
	}

	// For now, use minimal config to avoid filtering out metrics
	cfg := config.Default()
	// Skip config file loading for debug - we want to see all metrics
	// if configFile != "" {
	// 	cfg, err = config.Load(configFile)
	// 	if err != nil {
	// 		writer.Close()
	// 		return nil, fmt.Errorf("failed to load config: %w", err)
	// 	}
	// }

	return &Converter{
		writer: writer,
		config: cfg,
	}, nil
}

func (c *Converter) Close() error {
	return c.writer.Close()
}

func (c *Converter) GetWriter() *tsdb.Writer {
	return c.writer
}

func (c *Converter) ConvertFile(filename string) error {
	// Use Go parser directly for now (Java extractor has compilation issues)
	reader, err := gfs.NewStatArchiveReader(filename)
	if err != nil {
		return fmt.Errorf("failed to create StatArchive reader: %w", err)
	}
	defer reader.Close()
	return c.convertWithReader(reader, filename)
}

// Define interface for both readers
type StatReader interface {
	ReadArchive() error
	GetResourceTypes() map[int32]*gfs.ResourceType
	GetInstances() map[int32]*gfs.ResourceInstance
	GetArchiveInfo() map[string]interface{}
	Close() error
}

func (c *Converter) convertWithReader(reader StatReader, filename string) error {
	log.Printf("Parsing GFS file: %s", filename)
	if err := reader.ReadArchive(); err != nil {
		log.Printf("Warning: Archive parsing completed with errors: %v", err)
	}

	types := reader.GetResourceTypes()
	instances := reader.GetInstances()

	totalMetrics := 0
	for _, instance := range instances {
		resType, ok := types[instance.TypeID]
		if !ok {
			log.Printf("Warning: Unknown resource type %d for instance %s", instance.TypeID, instance.Name)
			continue
		}

		// Skip corrupted types/instances
		if !c.isValidResourceType(resType) || !c.isValidInstance(instance) {
			continue
		}

		// Iterate through all stats for this resource type
		for i, stat := range resType.Stats {
			statID := int32(i)
			
			// Check if we have data for this stat
			values, hasData := instance.Stats[statID]
			if !hasData || len(values) == 0 {
				continue
			}

			metricName := c.formatMetricName(resType.Name, stat.Name)
			
			// Use proper Prometheus labels as requested
			labels := map[string]string{
				"job":      "gfs-to-prometheus",
				"statType": resType.Name,
				"statName": instance.Name,
			}
			
			// Write ALL values for this stat, preserving original timestamps
			for i, sample := range values {
				value := c.convertToFloat64(sample.Value)
				
				// Use the original timestamp from the GFS file
				timestamp := sample.Timestamp
				
				if err := c.writer.WriteMetric(metricName, labels, value, timestamp); err != nil {
					log.Printf("Warning: Failed to write metric %s sample %d: %v", metricName, i, err)
					continue
				}
				totalMetrics++
			}
		}
	}

	if err := c.writer.Commit(); err != nil {
		return fmt.Errorf("failed to commit metrics: %w", err)
	}

	log.Printf("Converted %d metrics from %s", totalMetrics, filename)
	return nil
}

func (c *Converter) isValidResourceType(resType *gfs.ResourceType) bool {
	if len(resType.Name) == 0 || len(resType.Name) > 100 {
		return false
	}
	
	// Check for reasonable characters
	for _, r := range resType.Name {
		if r < 32 || r > 126 {
			return false
		}
	}
	
	return true
}

func (c *Converter) isValidInstance(instance *gfs.ResourceInstance) bool {
	if len(instance.Name) == 0 || len(instance.Name) > 200 {
		return false
	}
	
	// Check for reasonable characters (allow more flexibility for instance names)
	validChars := 0
	for _, r := range instance.Name {
		if r >= 32 && r <= 126 {
			validChars++
		}
	}
	
	// At least 80% of characters should be printable
	return float64(validChars)/float64(len(instance.Name)) >= 0.8
}

func (c *Converter) formatMetricName(resourceType, statName string) string {
	prefix := c.config.MetricPrefix
	if prefix == "" {
		prefix = "gemfire"
	}

	resourceType = strings.ToLower(strings.ReplaceAll(resourceType, " ", "_"))
	statName = strings.ToLower(strings.ReplaceAll(statName, " ", "_"))
	statName = strings.ReplaceAll(statName, "-", "_")

	return fmt.Sprintf("%s_%s_%s", prefix, resourceType, statName)
}

func (c *Converter) convertToFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}