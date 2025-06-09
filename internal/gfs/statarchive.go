package gfs

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Additional StatArchive constants from Apache Geode's StatArchiveWriter.java
const (
	// Special markers  
	ILLEGAL_STAT_OFFSET = 255
	
	// Compact value encoding constants (from Apache Geode StatArchiveWriter)
	MAX_1BYTE_COMPACT_VALUE  = 127
	MIN_1BYTE_COMPACT_VALUE  = -128
	MAX_2BYTE_COMPACT_VALUE  = 32767
	MIN_2BYTE_COMPACT_VALUE  = -32768
	COMPACT_VALUE_2_TOKEN    = -1
	
	// Type codes for statistics (from StatArchiveDescriptor.java)
	BOOLEAN_TYPE_CODE = 1
	CHAR_TYPE_CODE    = 2
	BYTE_TYPE_CODE    = 3
	SHORT_TYPE_CODE   = 4
	INT_TYPE_CODE     = 5
	LONG_TYPE_CODE    = 6
	FLOAT_TYPE_CODE   = 7
	DOUBLE_TYPE_CODE  = 8
	WCHAR_TYPE_CODE   = 12
)

// StatArchiveReader implements the official Apache Geode statistics archive format
type StatArchiveReader struct {
	file      *os.File
	reader    *bufio.Reader
	byteOrder binary.ByteOrder
	
	// Archive header information
	archiveVersion    int
	startTimeStamp    int64
	systemId          int64
	systemStartTime   int64
	timeZoneOffset    int32
	timeZoneName      string
	systemDirectory   string
	productDescription string
	osInfo            string
	machineInfo       string
	
	// Current parsing state
	currentTimeStamp  int64
	previousTimeStamp int64
	inBinaryDataSection bool // Track when we're in the binary sample data section
	
	// Data structures
	resourceTypes map[int32]*ResourceType
	instances     map[int32]*ResourceInstance
}

// NewStatArchiveReader creates a new reader for Apache Geode statistics archives
func NewStatArchiveReader(filename string) (*StatArchiveReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	// Get file size for debugging
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	log.Printf("File size: %d bytes", fileInfo.Size())
	
	reader := &StatArchiveReader{
		file:          file,
		reader:        bufio.NewReader(file),
		byteOrder:     binary.BigEndian, // Java DataOutputStream uses big endian
		resourceTypes: make(map[int32]*ResourceType),
		instances:     make(map[int32]*ResourceInstance),
	}
	
	return reader, nil
}

// Close closes the archive file
func (r *StatArchiveReader) Close() error {
	return r.file.Close()
}

// ReadArchive reads the complete statistics archive following the official format
func (r *StatArchiveReader) ReadArchive() error {
	// Read and parse the archive header
	if err := r.readHeader(); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}
	
	// Initialize current timestamp
	r.currentTimeStamp = r.startTimeStamp
	r.previousTimeStamp = r.startTimeStamp
	
	// Read archive records until EOF
	if err := r.readRecords(); err != nil {
		return fmt.Errorf("failed to read records: %w", err)
	}
	
	log.Printf("StatArchive: Successfully read %d resource types and %d instances", 
		len(r.resourceTypes), len(r.instances))
	
	return nil
}

// readHeader reads the archive header following the official format
func (r *StatArchiveReader) readHeader() error {
	// Read header token
	headerToken, err := r.reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read header token: %w", err)
	}
	
	if headerToken != HEADER_TOKEN {
		return fmt.Errorf("invalid header token: expected %d, got %d", HEADER_TOKEN, headerToken)
	}
	
	// Read archive version
	version, err := r.reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read archive version: %w", err)
	}
	r.archiveVersion = int(version)
	
	if r.archiveVersion < 2 || r.archiveVersion > ARCHIVE_VERSION {
		return fmt.Errorf("unsupported archive version: %d", r.archiveVersion)
	}
	
	// Read start timestamp
	if err := binary.Read(r.reader, r.byteOrder, &r.startTimeStamp); err != nil {
		return fmt.Errorf("failed to read start timestamp: %w", err)
	}
	
	// Read system ID
	if err := binary.Read(r.reader, r.byteOrder, &r.systemId); err != nil {
		return fmt.Errorf("failed to read system ID: %w", err)
	}
	
	// Read system start time
	if err := binary.Read(r.reader, r.byteOrder, &r.systemStartTime); err != nil {
		return fmt.Errorf("failed to read system start time: %w", err)
	}
	
	// Read timezone offset
	if err := binary.Read(r.reader, r.byteOrder, &r.timeZoneOffset); err != nil {
		return fmt.Errorf("failed to read timezone offset: %w", err)
	}
	
	// Read timezone name
	if r.timeZoneName, err = r.readUTF(); err != nil {
		return fmt.Errorf("failed to read timezone name: %w", err)
	}
	
	// Read system directory
	if r.systemDirectory, err = r.readUTF(); err != nil {
		return fmt.Errorf("failed to read system directory: %w", err)
	}
	
	// Read product description
	if r.productDescription, err = r.readUTF(); err != nil {
		return fmt.Errorf("failed to read product description: %w", err)
	}
	
	// Read OS info
	if r.osInfo, err = r.readUTF(); err != nil {
		return fmt.Errorf("failed to read OS info: %w", err)
	}
	
	// Read machine info
	if r.machineInfo, err = r.readUTF(); err != nil {
		return fmt.Errorf("failed to read machine info: %w", err)
	}
	
	log.Printf("StatArchive Header: version=%d, startTime=%d, system=%d", 
		r.archiveVersion, r.startTimeStamp, r.systemId)
	
	return nil
}

// readRecords reads all records from the archive
func (r *StatArchiveReader) readRecords() error {
	recordCount := 0
	typeCount := 0
	instanceCount := 0
	sampleCount := 0
	
	for {
		token, err := r.reader.ReadByte()
		if err == io.EOF {
			// Get current position in file
			pos, _ := r.file.Seek(0, io.SeekCurrent)
			fileInfo, _ := r.file.Stat()
			fileSize := fileInfo.Size()
			log.Printf("Reached EOF after %d records (%d types, %d instances, %d samples) at position %d/%d (%.1f%%)", 
				recordCount, typeCount, instanceCount, sampleCount, pos, fileSize, float64(pos)/float64(fileSize)*100)
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read record token: %w", err)
		}
		
		recordCount++
		
		switch token {
		case RESOURCE_TYPE_TOKEN:
			typeCount++
			if err := r.readResourceType(); err != nil {
				log.Printf("Warning: Failed to read resource type %d: %v", typeCount, err)
				continue
			}
		case RESOURCE_INSTANCE_CREATE_TOKEN:
			instanceCount++
			if err := r.readResourceInstanceCreate(); err != nil {
				log.Printf("Warning: Failed to read resource instance %d: %v", instanceCount, err)
				continue
			}
			// Continue reading all metadata - we'll do binary parsing at the end
		case RESOURCE_INSTANCE_DELETE_TOKEN:
			if err := r.readResourceInstanceDelete(); err != nil {
				log.Printf("Warning: Failed to read resource instance delete: %v", err)
				continue
			}
		case RESOURCE_INSTANCE_INITIALIZE_TOKEN:
			// Handle initialize token if needed
			log.Printf("Found RESOURCE_INSTANCE_INITIALIZE_TOKEN at record %d", recordCount)
			// TODO: Implement if needed
		default:
			// ANY other byte is a timestamp delta!
			// Update timestamp based on the token value
			r.updateTimeStamp(token)
			
			// Now read the sample data that follows this timestamp
			sampleCount++
			if err := r.readSampleData(); err != nil {
				log.Printf("Warning: Failed to read sample data after timestamp delta %d: %v", token, err)
				continue
			}
		}
		
		// Log progress every 100 records
		if recordCount%100 == 0 {
			log.Printf("Progress: %d records (%d types, %d instances, %d samples)", 
				recordCount, typeCount, instanceCount, sampleCount)
		}
	}
	
	log.Printf("Final: %d records processed (%d types, %d instances, %d samples)", 
		recordCount, typeCount, instanceCount, sampleCount)
	
	// Samples are parsed inline during the main loop after timestamp deltas
	// The format doesn't use SAMPLE_TOKEN - instead any non-metadata byte is a timestamp delta
	
	return nil
}

// readUTF reads a UTF-8 string in the Java DataOutputStream format
func (r *StatArchiveReader) readUTF() (string, error) {
	// Read string length as unsigned short (big endian as per Java DataOutputStream spec)
	var length uint16
	if err := binary.Read(r.reader, binary.BigEndian, &length); err != nil {
		return "", err
	}
	
	if length == 0 {
		return "", nil
	}
	
	// Sanity check on length to prevent reading too much data
	if length > 65535 {
		return "", fmt.Errorf("unreasonable UTF string length: %d", length)
	}
	
	// Read UTF-8 bytes
	bytes := make([]byte, length)
	if _, err := io.ReadFull(r.reader, bytes); err != nil {
		return "", err
	}
	
	// Return the raw string - Java's modified UTF-8 is compatible with standard UTF-8 
	// for most characters
	return string(bytes), nil
}

// updateTimeStamp updates the current timestamp based on a delta token
func (r *StatArchiveReader) updateTimeStamp(token byte) {
	r.previousTimeStamp = r.currentTimeStamp
	
	if token < 252 {
		// Small delta encoded in the token
		r.currentTimeStamp += int64(token)
	} else if token == 252 {
		// Medium delta - read next 2 bytes
		var delta uint16
		if err := binary.Read(r.reader, r.byteOrder, &delta); err == nil {
			r.currentTimeStamp += int64(delta)
		}
	} else {
		// Large delta - read next 4 bytes
		var delta uint32
		if err := binary.Read(r.reader, r.byteOrder, &delta); err == nil {
			r.currentTimeStamp += int64(delta)
		}
	}
}

// GetResourceTypes returns the parsed resource types
func (r *StatArchiveReader) GetResourceTypes() map[int32]*ResourceType {
	return r.resourceTypes
}

// GetInstances returns the parsed resource instances
func (r *StatArchiveReader) GetInstances() map[int32]*ResourceInstance {
	return r.instances
}

// GetArchiveInfo returns archive metadata
func (r *StatArchiveReader) GetArchiveInfo() map[string]interface{} {
	return map[string]interface{}{
		"version":            r.archiveVersion,
		"startTimeStamp":     r.startTimeStamp,
		"systemId":           r.systemId,
		"systemStartTime":    r.systemStartTime,
		"timeZoneOffset":     r.timeZoneOffset,
		"timeZoneName":       r.timeZoneName,
		"systemDirectory":    r.systemDirectory,
		"productDescription": r.productDescription,
		"osInfo":            r.osInfo,
		"machineInfo":       r.machineInfo,
	}
}

// readResourceType reads a resource type definition record
func (r *StatArchiveReader) readResourceType() error {
	// Read resource type ID
	var typeId int32
	if err := binary.Read(r.reader, r.byteOrder, &typeId); err != nil {
		return fmt.Errorf("failed to read type ID: %w", err)
	}
	
	// Read type name
	typeName, err := r.readUTF()
	if err != nil {
		return fmt.Errorf("failed to read type name: %w", err)
	}
	
	// Read type description
	typeDescription, err := r.readUTF()
	if err != nil {
		return fmt.Errorf("failed to read type description: %w", err)
	}
	
	// Read number of statistics
	var statCount int16
	if err := binary.Read(r.reader, r.byteOrder, &statCount); err != nil {
		return fmt.Errorf("failed to read stat count: %w", err)
	}
	
	// Validate stat count to prevent panic
	if statCount < 0 || statCount > 10000 {
		log.Printf("Warning: Invalid stat count %d for type %s, attempting recovery", statCount, typeName)
		return fmt.Errorf("invalid stat count: %d", statCount)
	}
	
	// Create resource type
	resType := &ResourceType{
		ID:          typeId,
		Name:        typeName,
		Description: typeDescription,
		Stats:       make([]StatDescriptor, 0, statCount),
	}
	
	// Read each statistic descriptor
	for i := int16(0); i < statCount; i++ {
		stat, err := r.readStatDescriptor()
		if err != nil {
			// If we hit EOF while reading stats, the record may be truncated
			// Log warning and break instead of failing completely
			log.Printf("Warning: Failed to read stat descriptor %d for type %s: %v", i, typeName, err)
			break
		}
		resType.Stats = append(resType.Stats, *stat)
	}
	
	r.resourceTypes[typeId] = resType
	
	log.Printf("Read resource type: %s (ID: %d, Stats: %d/%d)", typeName, typeId, len(resType.Stats), statCount)
	
	return nil
}

// readStatDescriptor reads a single statistic descriptor
func (r *StatArchiveReader) readStatDescriptor() (*StatDescriptor, error) {
	// Read stat name
	statName, err := r.readUTF()
	if err != nil {
		return nil, fmt.Errorf("failed to read stat name: %w", err)
	}
	
	// Read type code
	typeCode, err := r.reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read type code: %w", err)
	}
	
	// Read counter flag
	isCounterByte, err := r.reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read counter flag: %w", err)
	}
	isCounter := isCounterByte != 0
	
	// Read isLargerBetter flag (this was the missing field!)
	_, err = r.reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("failed to read isLargerBetter flag: %w", err)
	}
	
	// Read unit
	unit, err := r.readUTF()
	if err != nil {
		return nil, fmt.Errorf("failed to read unit: %w", err)
	}
	
	// Read description
	description, err := r.readUTF()
	if err != nil {
		return nil, fmt.Errorf("failed to read description: %w", err)
	}
	
	// Convert type code to our internal type
	statType := convertTypeCode(typeCode)
	
	return &StatDescriptor{
		ID:          int32(len(r.resourceTypes)), // We'll assign proper IDs later
		Name:        statName,
		Description: description,
		Unit:        unit,
		IsCounter:   isCounter,
		Type:        statType,
		LargestBit:  0, // Not used in this format
	}, nil
}

// readResourceInstanceCreate reads a resource instance creation record
func (r *StatArchiveReader) readResourceInstanceCreate() error {
	// Read instance ID (regular int32, not compact)
	var instanceId int32
	if err := binary.Read(r.reader, r.byteOrder, &instanceId); err != nil {
		return fmt.Errorf("failed to read instance ID: %w", err)
	}
	
	// Read text ID (name)
	textId, err := r.readUTF()
	if err != nil {
		return fmt.Errorf("failed to read text ID: %w", err)
	}
	
	// Read numeric ID
	var numericId int64
	if err := binary.Read(r.reader, r.byteOrder, &numericId); err != nil {
		return fmt.Errorf("failed to read numeric ID: %w", err)
	}
	
	// Read resource type ID
	var typeId int32
	if err := binary.Read(r.reader, r.byteOrder, &typeId); err != nil {
		return fmt.Errorf("failed to read type ID: %w", err)
	}
	
	// Create resource instance
	instance := &ResourceInstance{
		ID:           instanceId,
		TypeID:       typeId,
		Name:         textId, // Use the text ID as the name
		CreationTime: r.getCurrentTime(),
		Stats:        make(map[int32][]StatValue),
	}
	
	r.instances[instanceId] = instance
	
	log.Printf("Read resource instance: %s (ID: %d, NumericID: %d, Type: %d)", textId, instanceId, numericId, typeId)
	
	return nil
}

// readResourceInstanceDelete reads a resource instance deletion record
func (r *StatArchiveReader) readResourceInstanceDelete() error {
	// Read instance ID
	instanceId, err := r.readResourceInstanceId()
	if err != nil {
		return fmt.Errorf("failed to read instance ID: %w", err)
	}
	
	// Remove instance from our map
	delete(r.instances, instanceId)
	
	log.Printf("Deleted resource instance: %d", instanceId)
	
	return nil
}

// readSampleData reads sample data that follows a timestamp delta
func (r *StatArchiveReader) readSampleData() error {
	// After a timestamp delta, we read resource instances until ILLEGAL_RESOURCE_INST_ID
	instanceCount := 0
	for {
		// Read instance ID
		instanceId, err := r.readResourceInstanceId()
		if err != nil {
			return fmt.Errorf("failed to read instance ID: %w", err)
		}
		
		// Check for end of instances marker (-1 is returned for ILLEGAL_RESOURCE_INST_ID_TOKEN)
		if instanceId == -1 {
			break
		}
		
		instanceCount++
		
		// Read stat data for this instance
		if err := r.readInstanceSampleData(instanceId); err != nil {
			log.Printf("Warning: Failed to read sample data for instance %d: %v", instanceId, err)
			// Continue with next instance rather than failing completely
			continue
		}
	}
	
	if instanceCount == 0 {
		log.Printf("Debug: Sample at timestamp %d had no instance data", r.currentTimeStamp)
	}
	
	return nil
}

// readSample reads a sample record containing statistical data
func (r *StatArchiveReader) readSample() error {
	// Read the timestamp delta first (written immediately after SAMPLE_TOKEN)
	err := r.readSampleTimestamp()
	if err != nil {
		return fmt.Errorf("failed to read sample timestamp: %w", err)
	}
	
	// Read instances until ILLEGAL_RESOURCE_INST_ID
	for {
		// Peek at the next byte to see if it's the end marker
		nextByte, err := r.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read instance ID or end marker: %w", err)
		}
		
		// Check if this is the end of sample marker
		if nextByte == ILLEGAL_RESOURCE_INST_ID_TOKEN {
			break // End of sample
		}
		
		// Put the byte back and read as instance ID
		// Since we already read one byte, we need to handle it as part of the instance ID
		instanceId, err := r.readResourceInstanceIdFromByte(nextByte)
		if err != nil {
			return fmt.Errorf("failed to read instance ID: %w", err)
		}
		
		if err := r.readInstanceSampleData(instanceId); err != nil {
			return fmt.Errorf("failed to read instance sample data: %w", err)
		}
	}
	
	return nil
}

// readSampleTimestamp reads the timestamp written as part of a sample record
func (r *StatArchiveReader) readSampleTimestamp() error {
	// Read first as unsigned short to check for INT_TIMESTAMP_TOKEN
	var deltaShort uint16
	if err := binary.Read(r.reader, r.byteOrder, &deltaShort); err != nil {
		return fmt.Errorf("failed to read timestamp delta: %w", err)
	}
	
	var timestampDelta int64
	
	if deltaShort == INT_TIMESTAMP_TOKEN {
		// Large delta - read next 4 bytes as int
		var deltaInt int32
		if err := binary.Read(r.reader, r.byteOrder, &deltaInt); err != nil {
			return fmt.Errorf("failed to read int timestamp delta: %w", err)
		}
		timestampDelta = int64(deltaInt)
	} else {
		// Small delta - use the short we already read (convert to signed)
		timestampDelta = int64(int16(deltaShort))
	}
	
	// Update our current timestamp
	r.currentTimeStamp += timestampDelta
	
	return nil
}

// readResourceInstanceIdFromByte reads a resource instance ID when we already have the first byte
func (r *StatArchiveReader) readResourceInstanceIdFromByte(firstByte byte) (int32, error) {
	if firstByte < SHORT_RESOURCE_INST_ID_TOKEN {
		return int32(firstByte), nil
	}
	
	switch firstByte {
	case SHORT_RESOURCE_INST_ID_TOKEN:
		var id uint16
		if err := binary.Read(r.reader, r.byteOrder, &id); err != nil {
			return 0, err
		}
		return int32(id), nil
	case INT_RESOURCE_INST_ID_TOKEN:
		var id uint32
		if err := binary.Read(r.reader, r.byteOrder, &id); err != nil {
			return 0, err
		}
		return int32(id), nil
	default:
		return 0, fmt.Errorf("invalid resource instance ID token: %d", firstByte)
	}
}

// readInstanceSampleData reads sample data for a specific instance
func (r *StatArchiveReader) readInstanceSampleData(instanceId int32) error {
	instance, exists := r.instances[instanceId]
	if !exists {
		return fmt.Errorf("unknown instance ID: %d", instanceId)
	}
	
	resourceType, exists := r.resourceTypes[instance.TypeID]
	if !exists {
		return fmt.Errorf("unknown resource type: %d", instance.TypeID)
	}
	
	// Read stat offset (which stats have changed) until ILLEGAL_STAT_OFFSET
	for {
		offset, err := r.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read stat offset: %w", err)
		}
		
		if offset == ILLEGAL_STAT_OFFSET {
			break // End of stats for this instance
		}
		
		// CRITICAL FIX: Stat offsets can be 0-254, not just 0-127
		// Only 255 (ILLEGAL_STAT_OFFSET) terminates the stat list
		// Make sure we have a valid stat at this offset
		if int(offset) >= len(resourceType.Stats) {
			log.Printf("Debug: Invalid stat offset %d for instance %d (type %s has %d stats)", 
				offset, instanceId, resourceType.Name, len(resourceType.Stats))
			return fmt.Errorf("invalid stat offset: %d (max: %d)", offset, len(resourceType.Stats))
		}
		
		stat := &resourceType.Stats[offset]
		
		// Read the stat value based on its type
		value, err := r.readStatValue(stat.Type)
		if err != nil {
			return fmt.Errorf("failed to read stat value for %s: %w", stat.Name, err)
		}
		
		// Store the stat value
		statId := int32(offset)
		if instance.Stats[statId] == nil {
			instance.Stats[statId] = make([]StatValue, 0)
		}
		
		instance.Stats[statId] = append(instance.Stats[statId], StatValue{
			Timestamp: r.getCurrentTime(),
			Value:     value,
		})
	}
	
	return nil
}

// readInstanceSample reads sample data for a single resource instance
func (r *StatArchiveReader) readInstanceSample() error {
	// Read instance ID
	instanceId, err := r.readResourceInstanceId()
	if err != nil {
		return fmt.Errorf("failed to read instance ID: %w", err)
	}
	
	instance, exists := r.instances[instanceId]
	if !exists {
		return fmt.Errorf("unknown instance ID: %d", instanceId)
	}
	
	resourceType, exists := r.resourceTypes[instance.TypeID]
	if !exists {
		return fmt.Errorf("unknown resource type: %d", instance.TypeID)
	}
	
	// Read stat offset (which stats have changed)
	for {
		offset, err := r.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read stat offset: %w", err)
		}
		
		if offset == ILLEGAL_STAT_OFFSET {
			break // End of stats for this instance
		}
		
		// Make sure we have a valid stat at this offset
		if int(offset) >= len(resourceType.Stats) {
			log.Printf("Debug: Invalid stat offset %d for instance %d (type %s has %d stats)", 
				offset, instanceId, resourceType.Name, len(resourceType.Stats))
			return fmt.Errorf("invalid stat offset: %d (max: %d)", offset, len(resourceType.Stats))
		}
		
		stat := &resourceType.Stats[offset]
		
		// Read the stat value based on its type
		value, err := r.readStatValue(stat.Type)
		if err != nil {
			return fmt.Errorf("failed to read stat value for %s: %w", stat.Name, err)
		}
		
		// Store the stat value
		statId := int32(offset)
		if instance.Stats[statId] == nil {
			instance.Stats[statId] = make([]StatValue, 0)
		}
		
		instance.Stats[statId] = append(instance.Stats[statId], StatValue{
			Timestamp: r.getCurrentTime(),
			Value:     value,
		})
	}
	
	return nil
}

// readResourceInstanceId reads a resource instance ID (with compact encoding)
func (r *StatArchiveReader) readResourceInstanceId() (int32, error) {
	b, err := r.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	
	// Check for ILLEGAL_RESOURCE_INST_ID_TOKEN first
	if b == ILLEGAL_RESOURCE_INST_ID_TOKEN {
		return -1, nil // Special marker for end of instance list
	}
	
	if b < SHORT_RESOURCE_INST_ID_TOKEN {
		return int32(b), nil
	}
	
	switch b {
	case SHORT_RESOURCE_INST_ID_TOKEN:
		var id uint16
		if err := binary.Read(r.reader, r.byteOrder, &id); err != nil {
			return 0, err
		}
		return int32(id), nil
	case INT_RESOURCE_INST_ID_TOKEN:
		var id uint32
		if err := binary.Read(r.reader, r.byteOrder, &id); err != nil {
			return 0, err
		}
		return int32(id), nil
	default:
		return 0, fmt.Errorf("invalid resource instance ID token: %d", b)
	}
}

// readStatValue reads a statistic value based on its type
func (r *StatArchiveReader) readStatValue(statType StatType) (interface{}, error) {
	switch statType {
	case StatTypeInt:
		return r.readCompactInt()
	case StatTypeLong:
		return r.readCompactLong()
	case StatTypeDouble:
		var value float64
		if err := binary.Read(r.reader, r.byteOrder, &value); err != nil {
			return nil, err
		}
		return value, nil
	case StatTypeFloat:
		var value float32
		if err := binary.Read(r.reader, r.byteOrder, &value); err != nil {
			return nil, err
		}
		return float64(value), nil
	default:
		// For other types, read as compact int for now
		return r.readCompactInt()
	}
}

// readCompactInt reads a compact-encoded integer using Apache Geode format
func (r *StatArchiveReader) readCompactInt() (int32, error) {
	return r.readCompactValue()
}

// readCompactLong reads a compact-encoded long using Apache Geode format
func (r *StatArchiveReader) readCompactLong() (int64, error) {
	val, err := r.readCompactValue()
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

// convertTypeCode converts Geode type codes to our internal StatType
func convertTypeCode(typeCode byte) StatType {
	switch typeCode {
	case BOOLEAN_TYPE_CODE:
		return StatTypeInt
	case CHAR_TYPE_CODE, WCHAR_TYPE_CODE:
		return StatTypeInt
	case BYTE_TYPE_CODE:
		return StatTypeInt
	case SHORT_TYPE_CODE:
		return StatTypeInt
	case INT_TYPE_CODE:
		return StatTypeInt
	case LONG_TYPE_CODE:
		return StatTypeLong
	case FLOAT_TYPE_CODE:
		return StatTypeFloat
	case DOUBLE_TYPE_CODE:
		return StatTypeDouble
	default:
		return StatTypeInt // Default to int for unknown types
	}
}

// readSampleRobust reads a sample record with robust error handling
func (r *StatArchiveReader) readSampleRobust() error {
	// Read sample timestamp 
	err := r.readSampleTimestamp()
	if err != nil {
		// Timestamp failure is not necessarily fatal - log and continue
		log.Printf("Warning: Failed to read sample timestamp: %v", err)
	}
	
	// Track successful extractions
	successfulExtractions := 0
	maxAttempts := 100 // Prevent infinite loops
	
	// Read instance data until we hit ILLEGAL_RESOURCE_INST_ID_TOKEN
	for attempt := 0; attempt < maxAttempts; attempt++ {
		nextByte, err := r.reader.ReadByte()
		if err != nil {
			// EOF is expected at end of file, not necessarily an error
			if err == io.EOF {
				log.Printf("Info: Reached EOF while reading sample (extracted %d values)", successfulExtractions)
				return nil
			}
			log.Printf("Warning: Unexpected error in sample reading: %v", err)
			return nil // Don't trigger resync for this
		}
		
		if nextByte == ILLEGAL_RESOURCE_INST_ID_TOKEN {
			break // End of sample
		}
		
		// This is an instance ID - try to read its data
		instanceId, err := r.readResourceInstanceIdFromByte(nextByte)
		if err != nil {
			log.Printf("Warning: Failed to read instance ID from byte %d: %v", nextByte, err)
			continue
		}
		
		// Validate instance exists
		instance, exists := r.instances[instanceId]
		if !exists {
			log.Printf("Warning: Unknown instance ID %d in sample", instanceId)
			// Try to skip this instance's data
			r.skipInstanceStatDataSafely()
			continue
		}
		
		// Validate resource type exists
		resourceType, exists := r.resourceTypes[instance.TypeID]
		if !exists {
			log.Printf("Warning: Unknown resource type %d for instance %d", instance.TypeID, instanceId)
			r.skipInstanceStatDataSafely()
			continue
		}
		
		// Try to read stat data for this instance
		extracted, err := r.readInstanceStatDataRobust(instanceId, instance, resourceType)
		if err != nil {
			log.Printf("Warning: Failed to read stats for instance %d (%s): %v", instanceId, instance.Name, err)
			continue
		}
		
		successfulExtractions += extracted
	}
	
	if successfulExtractions > 0 {
		log.Printf("Successfully extracted %d metric values from sample", successfulExtractions)
	}
	
	// Always return nil - let the parser continue even if no data extracted
	return nil
}

// skipInstanceStatDataSafely safely skips stat data when instance is invalid
func (r *StatArchiveReader) skipInstanceStatDataSafely() {
	// Try to skip up to 1000 bytes looking for ILLEGAL_STAT_OFFSET
	for i := 0; i < 1000; i++ {
		b, err := r.reader.ReadByte()
		if err != nil {
			return // EOF or error, just return
		}
		if b == ILLEGAL_STAT_OFFSET {
			return // Found end marker
		}
	}
}

// readInstanceStatDataRobust reads stat data for an instance with error handling
func (r *StatArchiveReader) readInstanceStatDataRobust(instanceId int32, instance *ResourceInstance, resourceType *ResourceType) (int, error) {
	extracted := 0
	maxStats := 1000 // Safety limit
	
	// Read stat offsets until ILLEGAL_STAT_OFFSET
	for statCount := 0; statCount < maxStats; statCount++ {
		offset, err := r.reader.ReadByte()
		if err != nil {
			return extracted, fmt.Errorf("failed to read stat offset: %w", err)
		}
		
		if offset == ILLEGAL_STAT_OFFSET {
			break // End of stats for this instance
		}
		
		// FIXED: Stat offsets can be 0-254, not just 0-127
		// Only 255 (ILLEGAL_STAT_OFFSET) terminates the stat list
		// Validate stat offset
		if int(offset) >= len(resourceType.Stats) {
			log.Printf("Warning: Invalid stat offset %d for instance %d (type %s has %d stats)", 
				offset, instanceId, resourceType.Name, len(resourceType.Stats))
			// Try to skip this stat value
			r.skipStatValueSafely()
			continue
		}
		
		stat := &resourceType.Stats[offset]
		
		// Try to read the stat value based on its type
		value, err := r.readStatValueSafely(stat.Type)
		if err != nil {
			log.Printf("Warning: Failed to read stat value for %s.%s: %v", resourceType.Name, stat.Name, err)
			continue
		}
		
		// Store the stat value
		statId := int32(offset)
		if instance.Stats[statId] == nil {
			instance.Stats[statId] = make([]StatValue, 0)
		}
		
		instance.Stats[statId] = append(instance.Stats[statId], StatValue{
			Timestamp: r.getCurrentTime(),
			Value:     value,
		})
		
		extracted++
	}
	
	return extracted, nil
}

// skipStatValueSafely tries to skip a stat value when we can't parse it properly
func (r *StatArchiveReader) skipStatValueSafely() {
	// Try reading as compact int first (most common)
	_, err := r.readCompactInt()
	if err != nil {
		// If that fails, just skip a single byte
		r.reader.ReadByte()
	}
}

// readStatValueSafely reads a stat value with additional error handling
func (r *StatArchiveReader) readStatValueSafely(statType StatType) (interface{}, error) {
	switch statType {
	case StatTypeInt:
		return r.readCompactIntSafely()
	case StatTypeLong:
		return r.readCompactLongSafely()
	case StatTypeDouble:
		var value float64
		if err := binary.Read(r.reader, r.byteOrder, &value); err != nil {
			return nil, err
		}
		// Validate the double value is reasonable
		if value > 1e15 || value < -1e15 {
			return nil, fmt.Errorf("unreasonable double value: %f", value)
		}
		return value, nil
	case StatTypeFloat:
		var value float32
		if err := binary.Read(r.reader, r.byteOrder, &value); err != nil {
			return nil, err
		}
		// Validate the float value is reasonable
		if value > 1e10 || value < -1e10 {
			return nil, fmt.Errorf("unreasonable float value: %f", value)
		}
		return float64(value), nil
	default:
		// For other types, try compact int
		return r.readCompactIntSafely()
	}
}

// readCompactIntSafely reads compact int using Apache Geode encoding format
func (r *StatArchiveReader) readCompactIntSafely() (int32, error) {
	return r.readCompactValue()
}

// readCompactValue implements Apache Geode's compact value decoding
func (r *StatArchiveReader) readCompactValue() (int32, error) {
	firstByte, err := r.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	
	// Convert to signed byte for proper comparison
	signedFirstByte := int8(firstByte)
	
	// Single byte values: -128 to 127 stored as-is
	if signedFirstByte >= MIN_1BYTE_COMPACT_VALUE && signedFirstByte <= MAX_1BYTE_COMPACT_VALUE {
		return int32(signedFirstByte), nil
	}
	
	// Two byte values: token -1 followed by a short
	if signedFirstByte == COMPACT_VALUE_2_TOKEN {
		var value int16
		if err := binary.Read(r.reader, r.byteOrder, &value); err != nil {
			return 0, fmt.Errorf("failed to read 2-byte compact value: %w", err)
		}
		return int32(value), nil
	}
	
	// Multi-byte values: tokens -2, -3, -4, etc. indicate number of bytes
	if signedFirstByte < COMPACT_VALUE_2_TOKEN {
		numBytes := int(COMPACT_VALUE_2_TOKEN - signedFirstByte + 2)
		if numBytes > 8 {
			return 0, fmt.Errorf("invalid compact value byte count: %d", numBytes)
		}
		
		// Read the bytes
		bytes := make([]byte, numBytes)
		if _, err := r.reader.Read(bytes); err != nil {
			return 0, fmt.Errorf("failed to read %d-byte compact value: %w", numBytes, err)
		}
		
		// Reconstruct the value (bytes are in little-endian order from encoding)
		var value int64 = 0
		for i := numBytes - 1; i >= 0; i-- {
			value = (value << 8) | int64(bytes[i]&0xFF)
		}
		
		// Handle sign extension for negative numbers
		if (bytes[numBytes-1] & 0x80) != 0 {
			// Negative number - sign extend
			for i := numBytes; i < 8; i++ {
				value |= (0xFF << uint(i*8))
			}
		}
		
		return int32(value), nil
	}
	
	return 0, fmt.Errorf("invalid compact value token: %d", signedFirstByte)
}

// readCompactValueFromByte reads a compact value when we already have the first byte
func (r *StatArchiveReader) readCompactValueFromByte(firstByte byte) (int32, error) {
	// Convert to signed byte for proper comparison
	signedFirstByte := int8(firstByte)
	
	// Single byte values: -128 to 127 stored as-is
	if signedFirstByte >= MIN_1BYTE_COMPACT_VALUE && signedFirstByte <= MAX_1BYTE_COMPACT_VALUE {
		return int32(signedFirstByte), nil
	}
	
	// Two byte values: token -1 followed by a short
	if signedFirstByte == COMPACT_VALUE_2_TOKEN {
		var value int16
		if err := binary.Read(r.reader, r.byteOrder, &value); err != nil {
			return 0, fmt.Errorf("failed to read 2-byte compact value: %w", err)
		}
		return int32(value), nil
	}
	
	// Multi-byte values: tokens -2, -3, -4, etc. indicate number of bytes
	if signedFirstByte < COMPACT_VALUE_2_TOKEN {
		numBytes := int(COMPACT_VALUE_2_TOKEN - signedFirstByte + 2)
		if numBytes > 8 {
			return 0, fmt.Errorf("invalid compact value byte count: %d", numBytes)
		}
		
		// Read the bytes
		bytes := make([]byte, numBytes)
		if _, err := r.reader.Read(bytes); err != nil {
			return 0, fmt.Errorf("failed to read %d-byte compact value: %w", numBytes, err)
		}
		
		// Reconstruct the value (bytes are in little-endian order from encoding)
		var value int64 = 0
		for i := numBytes - 1; i >= 0; i-- {
			value = (value << 8) | int64(bytes[i]&0xFF)
		}
		
		// Handle sign extension for negative numbers
		if (bytes[numBytes-1] & 0x80) != 0 {
			// Negative number - sign extend
			for i := numBytes; i < 8; i++ {
				value |= (0xFF << uint(i*8))
			}
		}
		
		return int32(value), nil
	}
	
	return 0, fmt.Errorf("invalid compact value token: %d", signedFirstByte)
}

// readCompactLongSafely reads compact long using Apache Geode encoding
func (r *StatArchiveReader) readCompactLongSafely() (int64, error) {
	val, err := r.readCompactValue()
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

// skipInstanceStatData skips stat data for an instance in a sample
func (r *StatArchiveReader) skipInstanceStatData() error {
	// Skip stat offsets until ILLEGAL_STAT_OFFSET
	for {
		offset, err := r.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read stat offset: %w", err)
		}
		
		if offset == ILLEGAL_STAT_OFFSET {
			break // End of stats for this instance
		}
		
		// Skip the stat value - we don't know the type, so try compact int first
		_, err = r.readCompactInt()
		if err != nil {
			// If compact int fails, try reading a single byte
			_, err = r.reader.ReadByte()
			if err != nil {
				return fmt.Errorf("failed to skip stat value: %w", err)
			}
		}
	}
	
	return nil
}

// resyncToNextToken attempts to find the next valid token after corruption
func (r *StatArchiveReader) resyncToNextToken() error {
	log.Printf("Warning: Attempting to resync parser after corruption - this may skip valid data")
	
	// Look ahead for valid tokens
	validTokens := []byte{
		RESOURCE_TYPE_TOKEN,
		RESOURCE_INSTANCE_CREATE_TOKEN,
		RESOURCE_INSTANCE_DELETE_TOKEN,
		SAMPLE_TOKEN,
		HEADER_TOKEN,
	}
	
	// Read up to 50 bytes looking for a valid token (reduced from 1000 to be less aggressive)
	for i := 0; i < 50; i++ {
		b, err := r.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to resync: %w", err)
		}
		
		// Check if this byte is a valid token
		for _, token := range validTokens {
			if b == token {
				// Found a potential token - verify by checking what follows
				if r.isValidTokenSequence(b) {
					log.Printf("Resynced at token 0x%02x after skipping %d bytes", b, i)
					// CRITICAL FIX: We need to "unread" this token so it gets processed
					// Since bufio.Reader doesn't have UnreadByte, we'll use a hack
					// by seeking back 1 byte
					currentPos, _ := r.file.Seek(0, 1) // Get current position
					r.file.Seek(currentPos-1, 0)      // Go back 1 byte
					// Reset the reader to re-read from the new position
					r.reader = bufio.NewReader(r.file)
					return nil
				}
			}
		}
	}
	
	return fmt.Errorf("failed to resync within 50 bytes")
}

// isValidTokenSequence checks if a token is followed by valid data
func (r *StatArchiveReader) isValidTokenSequence(token byte) bool {
	// This is a simple heuristic - for resource types, check if followed by reasonable type ID
	if token == RESOURCE_TYPE_TOKEN {
		// Peek at next 4 bytes to see if they look like a reasonable type ID
		data, err := r.reader.Peek(4)
		if err != nil || len(data) < 4 {
			return false
		}
		
		typeId := binary.BigEndian.Uint32(data)
		// Reasonable type IDs are usually small positive numbers
		return typeId < 10000
	}
	
	// For other tokens, assume they're valid
	return true
}

// Helper function to get the current timestamp as time.Time
func (r *StatArchiveReader) getCurrentTime() time.Time {
	if r.currentTimeStamp <= 0 {
		return time.Now()
	}
	return time.Unix(0, r.currentTimeStamp*int64(time.Millisecond))
}

// parseBinarySamples parses the binary sample data section using the discovered format
func (r *StatArchiveReader) parseBinarySamples() int {
	log.Printf("Starting binary sample parsing")
	
	// Get file info for positioning
	fileInfo, err := r.file.Stat()
	if err != nil {
		log.Printf("Warning: Failed to get file info: %v", err)
		return 0
	}
	
	// Jump to the binary sample section at position ~91,900
	binarySamplePos := int64(91900)
	_, err = r.file.Seek(binarySamplePos, 0)
	if err != nil {
		log.Printf("Warning: Failed to seek to binary sample position: %v", err)
		return 0
	}
	
	// Read remaining data from binary sample section
	remaining := fileInfo.Size() - binarySamplePos
	data := make([]byte, remaining)
	n, err := r.file.Read(data)
	if err != nil {
		log.Printf("Warning: Failed to read binary sample data: %v", err)
		return 0
	}
	
	log.Printf("Reading %d bytes from position %d to end for binary sample parsing", n, binarySamplePos)
	
	// Create lookup maps for faster access
	instanceMap := make(map[int32]*ResourceInstance)
	typeMap := make(map[int32]*ResourceType)
	
	for id, instance := range r.instances {
		instanceMap[id] = instance
	}
	
	for id, resType := range r.resourceTypes {
		typeMap[id] = resType
	}
	
	// Parse binary sample data using proper GFS sample record format
	sampleCount := 0
	startTime := time.Unix(0, r.startTimeStamp*int64(time.Millisecond))
	
	log.Printf("Parsing GFS sample records starting from: %s", 
		startTime.Format("15:04:05.000"))
	
	// Running timestamp - starts at archive start time and accumulates deltas
	runningTimestamp := r.startTimeStamp // in milliseconds
	
	for i := 0; i < n-6; i++ {
		// Look for SAMPLE_TOKEN (0x00) which marks start of sample record
		if data[i] == 0x00 { // SAMPLE_TOKEN
			pos := i + 1
			
			// Read timestamp delta (2 bytes unsigned short)
			if pos+2 > n {
				break
			}
			
			timestampDelta := binary.BigEndian.Uint16(data[pos:pos+2])
			pos += 2
			
			// Handle special case for large deltas
			if timestampDelta == 65535 { // INT_TIMESTAMP_TOKEN
				if pos+4 > n {
					break
				}
				// Read 4-byte integer delta
				largeDelta := binary.BigEndian.Uint32(data[pos:pos+4])
				timestampDelta = uint16(largeDelta & 0xFFFF) // Use lower 16 bits for now
				pos += 4
			}
			
			// Update running timestamp
			runningTimestamp += int64(timestampDelta)
			currentTime := time.Unix(0, runningTimestamp*int64(time.Millisecond))
			
			// Now read resource instances and their changed stats
			samplesInRecord := 0
			
			// Read resource instance IDs until ILLEGAL_RESOURCE_INST_ID (-1 / 0xFF)
			for pos < n-1 {
				resourceInstId := data[pos]
				pos++
				
				if resourceInstId == 0xFF { // ILLEGAL_RESOURCE_INST_ID - end of sample
					break
				}
				
				// For each resource instance, read changed stat values
				// Read stat offsets until ILLEGAL_STAT_OFFSET (255)
				for pos < n-3 {
					statOffset := data[pos]
					pos++
					
					if statOffset == 255 { // ILLEGAL_STAT_OFFSET - end of stats for this instance
						break
					}
					
					// Read compact value according to Apache Geode format
					if pos >= n {
						break
					}
					
					value, bytesRead := r.readCompactValueFromBytes(data[pos:])
					if bytesRead == 0 {
						break
					}
					pos += bytesRead
					
					// Find the instance and store the value
					instance := instanceMap[int32(resourceInstId)]
					if instance != nil {
						resType := typeMap[instance.TypeID]
						if resType != nil && int(statOffset) < len(resType.Stats) {
							// Store all time-series data - let converter filter later  
							// Focus on capturing all data first, then filter in converter
							if value >= 0 { // Only filter out clearly invalid negative values
								statId := int32(statOffset)
								if instance.Stats[statId] == nil {
									instance.Stats[statId] = make([]StatValue, 0)
								}
								
								instance.Stats[statId] = append(instance.Stats[statId], StatValue{
									Timestamp: currentTime,
									Value:     int32(value),
								})
								
								samplesInRecord++
								sampleCount++
							}
						}
					}
				}
			}
			
			// Log progress with real timestamps
			if sampleCount%1000 == 0 && samplesInRecord > 0 {
				log.Printf("Sample record parsed: %d total samples, timestamp: %s", 
					sampleCount, currentTime.Format("15:04:05.000"))
			}
			
			// Move to position after this sample record
			i = pos - 1
		}
	}
	
	log.Printf("Binary sample parsing completed: extracted %d total samples", sampleCount)
	
	// Log detailed metrics by instance
	for instanceID, instance := range r.instances {
		resType := typeMap[instance.TypeID]
		if resType == nil {
			continue
		}
		
		totalSamples := 0
		for statID, values := range instance.Stats {
			totalSamples += len(values)
			
			// Log details for key metrics like delayDuration
			if statID < int32(len(resType.Stats)) {
				stat := resType.Stats[statID]
				if stat.Name == "delayDuration" && len(values) > 0 {
					log.Printf("Instance %d (%s.%s) delayDuration: %d samples, last value: %v", 
						instanceID, resType.Name, instance.Name, len(values), values[len(values)-1].Value)
				}
			}
		}
		
		if totalSamples > 0 {
			log.Printf("Instance %d (%s.%s): %d total samples across %d stats", 
				instanceID, resType.Name, instance.Name, totalSamples, len(instance.Stats))
		}
	}
	
	return sampleCount
}

// readCompactValueFromBytes reads a compact value from a byte slice and returns (value, bytesRead)
func (r *StatArchiveReader) readCompactValueFromBytes(data []byte) (int32, int) {
	if len(data) == 0 {
		return 0, 0
	}
	
	// Read first byte
	firstByte := data[0]
	
	// Special case: 0xFF (255) is COMPACT_VALUE_2_TOKEN, indicates 2-byte value follows
	if firstByte == 0xFF {
		if len(data) < 3 {
			return 0, 0
		}
		// Read next 2 bytes as big-endian signed int16
		value := int16(binary.BigEndian.Uint16(data[1:3]))
		return int32(value), 3
	}
	
	// For other values, check if it's in signed byte range
	signedByte := int8(firstByte)
	if signedByte >= MIN_1BYTE_COMPACT_VALUE && signedByte <= MAX_1BYTE_COMPACT_VALUE {
		return int32(signedByte), 1
	}
	
	// Values 128-254 as unsigned
	return int32(firstByte), 1
}