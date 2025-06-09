package cluster

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/4n3w/gfs-to-prometheus/internal/converter"
	"github.com/4n3w/gfs-to-prometheus/internal/gfs"
)

// ClusterConverter wraps the regular converter to add cluster-specific labels
type ClusterConverter struct {
	Converter   *converter.Converter
	ClusterName string
	NodeName    string
	NodeType    string
}

func (cc *ClusterConverter) ConvertFile(filename string) error {
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

		// Create cluster-aware labels
		labels := cc.createLabels(resType.Name, instance.Name)

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

			metricName := cc.formatMetricName(resType.Name, statDesc.Name)
			
			for _, sv := range values {
				value := cc.convertToFloat64(sv.Value)
				if err := cc.writeMetric(metricName, labels, value, sv.Timestamp); err != nil {
					return fmt.Errorf("failed to write metric: %w", err)
				}
				totalMetrics++
			}
		}
	}

	if err := cc.Converter.Close(); err != nil {
		return fmt.Errorf("failed to commit metrics: %w", err)
	}

	log.Printf("Converted %d metrics from %s (cluster=%s, node=%s)", 
		totalMetrics, filename, cc.ClusterName, cc.NodeName)
	return nil
}

func (cc *ClusterConverter) createLabels(resourceType, instanceName string) map[string]string {
	labels := map[string]string{
		"cluster":       cc.ClusterName,
		"node":          cc.NodeName,
		"node_type":     cc.NodeType,
		"resource_type": resourceType,
		"instance":      instanceName,
	}

	// Add deployment environment if we can infer it
	if env := cc.inferEnvironment(); env != "" {
		labels["environment"] = env
	}

	return labels
}

func (cc *ClusterConverter) inferEnvironment() string {
	// Try to infer environment from cluster name
	clusterLower := strings.ToLower(cc.ClusterName)
	
	if strings.Contains(clusterLower, "prod") || strings.Contains(clusterLower, "production") {
		return "production"
	}
	if strings.Contains(clusterLower, "dev") || strings.Contains(clusterLower, "development") {
		return "development"
	}
	if strings.Contains(clusterLower, "test") || strings.Contains(clusterLower, "testing") {
		return "testing"
	}
	if strings.Contains(clusterLower, "stag") || strings.Contains(clusterLower, "staging") {
		return "staging"
	}
	
	return ""
}

func (cc *ClusterConverter) formatMetricName(resourceType, statName string) string {
	prefix := "gemfire" // Could be configurable
	
	resourceType = strings.ToLower(strings.ReplaceAll(resourceType, " ", "_"))
	statName = strings.ToLower(strings.ReplaceAll(statName, " ", "_"))
	statName = strings.ReplaceAll(statName, "-", "_")

	return fmt.Sprintf("%s_%s_%s", prefix, resourceType, statName)
}

func (cc *ClusterConverter) convertToFloat64(value interface{}) float64 {
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

func (cc *ClusterConverter) writeMetric(name string, labels map[string]string, value float64, timestamp time.Time) error {
	// Write directly to TSDB with cluster labels
	writer := cc.Converter.GetWriter()
	return writer.WriteMetric(name, labels, value, timestamp)
}