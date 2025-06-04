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

	cfg := config.Default()
	if configFile != "" {
		cfg, err = config.Load(configFile)
		if err != nil {
			writer.Close()
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	return &Converter{
		writer: writer,
		config: cfg,
	}, nil
}

func (c *Converter) Close() error {
	return c.writer.Close()
}

func (c *Converter) ConvertFile(filename string) error {
	parser, err := gfs.NewGeodeParser(filename)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	defer parser.Close()

	log.Printf("Parsing GFS file: %s", filename)
	if err := parser.ParseGeode(); err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	types := parser.GetTypes()
	instances := parser.GetInstances()

	totalMetrics := 0
	for _, instance := range instances {
		resType, ok := types[instance.TypeID]
		if !ok {
			log.Printf("Warning: Unknown resource type %d for instance %s", instance.TypeID, instance.Name)
			continue
		}

		labels := map[string]string{
			"resource_type": resType.Name,
			"instance":      instance.Name,
		}

		for statID, values := range instance.Stats {
			var statDesc *gfs.StatDescriptor
			for _, s := range resType.Stats {
				if s.ID == statID {
					statDesc = &s
					break
				}
			}

			if statDesc == nil {
				continue
			}

			metricName := c.formatMetricName(resType.Name, statDesc.Name)
			
			for _, sv := range values {
				value := c.convertToFloat64(sv.Value)
				if err := c.writer.WriteMetric(metricName, labels, value, sv.Timestamp); err != nil {
					return fmt.Errorf("failed to write metric: %w", err)
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