package gfs

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// JavaExtractedData represents the structure returned by the Java extractor
type JavaExtractedData struct {
	ArchiveStartTime int64          `json:"archiveStartTime"`
	TotalSamples     int            `json:"totalSamples"`
	ResourceTypes    []ResourceType `json:"resourceTypes"`
	Instances        []JavaInstance `json:"instances"`
}

type JavaInstance struct {
	ID      int32        `json:"id"`
	TypeID  int32        `json:"typeId"`
	Name    string       `json:"name"`
	Samples []JavaSample `json:"samples"`
}

type JavaSample struct {
	StatID    int32 `json:"statId"`
	Timestamp int64 `json:"timestamp"` // milliseconds since epoch
	Value     interface{} `json:"value"`
}

// JavaStatArchiveReader uses Java libraries to parse GFS files correctly
type JavaStatArchiveReader struct {
	filename string
	data     *JavaExtractedData
}

func NewJavaStatArchiveReader(filename string) (*JavaStatArchiveReader, error) {
	return &JavaStatArchiveReader{
		filename: filename,
	}, nil
}

func (r *JavaStatArchiveReader) ReadArchive() error {
	// Build the Java extractor if needed
	if err := r.buildJavaExtractor(); err != nil {
		return fmt.Errorf("failed to build Java extractor: %w", err)
	}
	
	// Create temporary output file
	outputFile := filepath.Join(os.TempDir(), "gfs_extracted.json")
	defer os.Remove(outputFile)
	
	// Run Java extractor with proper classpath
	cmd := exec.Command("java", "-cp", "java-extractor/lib/*:java-extractor/build/stat-extractor.jar", 
						"StatExtractor", r.filename, outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Java extractor failed: %w\nOutput: %s", err, string(output))
	}
	
	// Read extracted data
	jsonData, err := os.ReadFile(outputFile)
	if err != nil {
		return fmt.Errorf("failed to read extracted data: %w", err)
	}
	
	// Parse JSON
	r.data = &JavaExtractedData{}
	if err := json.Unmarshal(jsonData, r.data); err != nil {
		return fmt.Errorf("failed to parse extracted data: %w", err)
	}
	
	return nil
}

func (r *JavaStatArchiveReader) buildJavaExtractor() error {
	// Check if JAR already exists
	jarPath := "java-extractor/build/stat-extractor.jar"
	if _, err := os.Stat(jarPath); err == nil {
		return nil // Already built
	}
	
	// Build with our custom build script
	cmd := exec.Command("./build.sh")
	cmd.Dir = "java-extractor"
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Java build failed: %w\nOutput: %s", err, string(output))
	}
	
	return nil
}

func (r *JavaStatArchiveReader) GetResourceTypes() map[int32]*ResourceType {
	if r.data == nil {
		return make(map[int32]*ResourceType)
	}
	
	types := make(map[int32]*ResourceType)
	for _, resType := range r.data.ResourceTypes {
		types[resType.ID] = &resType
	}
	return types
}

func (r *JavaStatArchiveReader) GetInstances() map[int32]*ResourceInstance {
	if r.data == nil {
		return make(map[int32]*ResourceInstance)
	}
	
	instances := make(map[int32]*ResourceInstance)
	for _, javaInstance := range r.data.Instances {
		instance := &ResourceInstance{
			ID:           javaInstance.ID,
			TypeID:       javaInstance.TypeID,
			Name:         javaInstance.Name,
			CreationTime: time.Unix(0, r.data.ArchiveStartTime*int64(time.Millisecond)),
			Stats:        make(map[int32][]StatValue),
		}
		
		// Convert samples to StatValue format
		for _, sample := range javaInstance.Samples {
			timestamp := time.Unix(0, sample.Timestamp*int64(time.Millisecond))
			statValue := StatValue{
				Timestamp: timestamp,
				Value:     sample.Value,
			}
			
			instance.Stats[sample.StatID] = append(instance.Stats[sample.StatID], statValue)
		}
		
		instances[javaInstance.ID] = instance
	}
	
	return instances
}

func (r *JavaStatArchiveReader) GetArchiveInfo() map[string]interface{} {
	if r.data == nil {
		return make(map[string]interface{})
	}
	
	return map[string]interface{}{
		"startTimeStamp": r.data.ArchiveStartTime,
		"totalSamples":   r.data.TotalSamples,
	}
}

func (r *JavaStatArchiveReader) Close() error {
	// Nothing to close for Java extractor approach
	return nil
}