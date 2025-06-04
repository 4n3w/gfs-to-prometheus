package gfs

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// Geode statistics archive constants based on StatArchiveWriter.java
const (
	// Tokens
	HEADER_TOKEN                     = 77
	SAMPLE_TOKEN                     = 0
	RESOURCE_TYPE_TOKEN              = 1
	RESOURCE_INSTANCE_CREATE_TOKEN   = 2
	RESOURCE_INSTANCE_DELETE_TOKEN   = 3
	RESOURCE_INSTANCE_INITIALIZE_TOKEN = 4
	
	// Compact value tokens
	COMPACT_VALUE_2_TOKEN = -128
	COMPACT_VALUE_3_TOKEN = -127
	COMPACT_VALUE_4_TOKEN = -126
	COMPACT_VALUE_5_TOKEN = -125
	COMPACT_VALUE_6_TOKEN = -124
	COMPACT_VALUE_7_TOKEN = -123
	COMPACT_VALUE_8_TOKEN = -122
	
	// Resource ID tokens
	SHORT_RESOURCE_INST_ID_TOKEN = 253
	INT_RESOURCE_INST_ID_TOKEN   = 254
	ILLEGAL_RESOURCE_INST_ID_TOKEN = 255
	
	// Timestamp tokens
	INT_TIMESTAMP_TOKEN = 65535
	
	// Type codes
	BOOLEAN_CODE = 1
	CHAR_CODE    = 2
	BYTE_CODE    = 3
	SHORT_CODE   = 4
	INT_CODE     = 5
	LONG_CODE    = 6
	FLOAT_CODE   = 7
	DOUBLE_CODE  = 8
	WCHAR_CODE   = 12
	
	// Archive version
	ARCHIVE_VERSION = 4
)

type GeodeParser struct {
	file         *os.File
	reader       *bufio.Reader
	byteOrder    binary.ByteOrder
	
	// Header information
	version       int
	startTime     int64
	systemID      int64
	systemStart   int64
	timeZoneOffset int32
	timeZoneName  string
	systemDir     string
	productDesc   string
	osInfo        string
	machineInfo   string
	
	// Current state
	currentTime   int64
	resourceTypes map[int]*ResourceType
	instances     map[int]*ResourceInstance
}

func NewGeodeParser(filename string) (*Parser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	// Create basic parser
	p := &Parser{
		file:      file,
		reader:    file,
		byteOrder: binary.BigEndian,
		types:     make(map[int32]*ResourceType),
		instances: make(map[int32]*ResourceInstance),
	}

	return p, nil
}

func (gp *GeodeParser) parseHeader() error {
	// Read header token
	token, err := gp.reader.ReadByte()
	if err != nil {
		return err
	}
	if token != HEADER_TOKEN {
		return fmt.Errorf("expected header token %d, got %d", HEADER_TOKEN, token)
	}

	// Read archive version (appears to be little endian in this file)
	var versionBytes [4]byte
	if _, err := io.ReadFull(gp.reader, versionBytes[:]); err != nil {
		return err
	}
	
	// Try little endian first based on the hex dump
	version := int(binary.LittleEndian.Uint32(versionBytes[:]))
	if version > 10 {
		// If too large, try big endian
		version = int(binary.BigEndian.Uint32(versionBytes[:]))
		gp.byteOrder = binary.BigEndian
	} else {
		gp.byteOrder = binary.LittleEndian
	}
	
	gp.version = version

	// Read timestamps
	gp.startTime, err = gp.readLong()
	if err != nil {
		return err
	}

	gp.systemID, err = gp.readLong()
	if err != nil {
		return err
	}

	gp.systemStart, err = gp.readLong()
	if err != nil {
		return err
	}

	// Read timezone info
	gp.timeZoneOffset, err = gp.readInt32()
	if err != nil {
		return err
	}

	gp.timeZoneName, err = gp.readUTF()
	if err != nil {
		return err
	}

	gp.systemDir, err = gp.readUTF()
	if err != nil {
		return err
	}

	gp.productDesc, err = gp.readUTF()
	if err != nil {
		return err
	}

	gp.osInfo, err = gp.readUTF()
	if err != nil {
		return err
	}

	gp.machineInfo, err = gp.readUTF()
	if err != nil {
		return err
	}

	return nil
}

func (gp *GeodeParser) parseRecords(p *Parser) error {
	for {
		token, err := gp.reader.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch token {
		case RESOURCE_TYPE_TOKEN:
			if err := gp.parseResourceType(p); err != nil {
				return fmt.Errorf("failed to parse resource type: %w", err)
			}
		case RESOURCE_INSTANCE_CREATE_TOKEN:
			if err := gp.parseResourceInstanceCreate(p); err != nil {
				return fmt.Errorf("failed to parse resource instance: %w", err)
			}
		case SAMPLE_TOKEN:
			if err := gp.parseSample(p); err != nil {
				return fmt.Errorf("failed to parse sample: %w", err)
			}
		default:
			// Handle timestamp delta
			delta := gp.decodeTimestamp(token)
			gp.currentTime += delta
		}
	}
	return nil
}

func (gp *GeodeParser) decodeTimestamp(token byte) int64 {
	if token < 252 {
		// Small delta encoded in the token itself
		return int64(token)
	}
	
	// Larger deltas require reading more bytes
	switch token {
	case SHORT_RESOURCE_INST_ID_TOKEN:
		// This shouldn't happen for timestamps, but handle it
		return 0
	default:
		// For now, assume it's a small delta
		return int64(token)
	}
}

func (gp *GeodeParser) parseResourceType(p *Parser) error {
	// Read resource type ID
	typeID, err := gp.readInt()
	if err != nil {
		return err
	}

	// Read resource type name
	typeName, err := gp.readUTF()
	if err != nil {
		return err
	}

	// Read description
	description, err := gp.readUTF()
	if err != nil {
		return err
	}

	// Read number of statistics
	statCount, err := gp.readShort()
	if err != nil {
		return err
	}

	resType := &ResourceType{
		ID:          int32(typeID),
		Name:        typeName,
		Description: description,
		Stats:       make([]StatDescriptor, 0, statCount),
	}

	// Read each statistic descriptor
	for i := 0; i < int(statCount); i++ {
		statName, err := gp.readUTF()
		if err != nil {
			return err
		}

		typeCode, err := gp.reader.ReadByte()
		if err != nil {
			return err
		}

		isCounter, err := gp.reader.ReadByte()
		if err != nil {
			return err
		}

		_, err = gp.reader.ReadByte() // largestBit - not used for now
		if err != nil {
			return err
		}

		unit, err := gp.readUTF()
		if err != nil {
			return err
		}

		desc, err := gp.readUTF()
		if err != nil {
			return err
		}

		stat := StatDescriptor{
			ID:          int32(i),
			Name:        statName,
			Description: desc,
			Unit:        unit,
			IsCounter:   isCounter != 0,
		}

		// Map type code to our enum
		switch typeCode {
		case BOOLEAN_CODE, BYTE_CODE, SHORT_CODE, INT_CODE, CHAR_CODE:
			stat.Type = StatTypeInt
		case LONG_CODE:
			stat.Type = StatTypeLong
		case FLOAT_CODE, DOUBLE_CODE:
			stat.Type = StatTypeDouble
		default:
			stat.Type = StatTypeInt
		}

		resType.Stats = append(resType.Stats, stat)
	}

	p.types[int32(typeID)] = resType
	gp.resourceTypes[typeID] = resType

	return nil
}

func (gp *GeodeParser) parseResourceInstanceCreate(p *Parser) error {
	// Read resource instance ID
	instID, err := gp.readResourceID()
	if err != nil {
		return err
	}

	// Read resource name
	name, err := gp.readUTF()
	if err != nil {
		return err
	}

	// Read resource type ID
	typeID, err := gp.readInt()
	if err != nil {
		return err
	}

	instance := &ResourceInstance{
		ID:           int32(instID),
		TypeID:       int32(typeID),
		Name:         name,
		CreationTime: time.Unix(0, gp.currentTime*int64(time.Millisecond)),
		Stats:        make(map[int32][]StatValue),
	}

	p.instances[int32(instID)] = instance
	gp.instances[instID] = instance

	return nil
}

func (gp *GeodeParser) parseSample(p *Parser) error {
	// Read number of samples
	sampleCount, err := gp.readShort()
	if err != nil {
		return err
	}

	for i := 0; i < int(sampleCount); i++ {
		// Read resource instance ID
		instID, err := gp.readResourceID()
		if err != nil {
			return err
		}

		instance, ok := gp.instances[instID]
		if !ok {
			return fmt.Errorf("unknown resource instance: %d", instID)
		}

		resType, ok := gp.resourceTypes[int(instance.TypeID)]
		if !ok {
			return fmt.Errorf("unknown resource type: %d", instance.TypeID)
		}

		// Read stat values
		statCount := len(resType.Stats)
		for j := 0; j < statCount; j++ {
			stat := &resType.Stats[j]
			
			// Read the value based on type
			var value interface{}
			switch stat.Type {
			case StatTypeInt:
				v, err := gp.readCompactValue()
				if err != nil {
					return err
				}
				value = int32(v)
			case StatTypeLong:
				v, err := gp.readCompactValue()
				if err != nil {
					return err
				}
				value = v
			case StatTypeDouble:
				v, err := gp.readDouble()
				if err != nil {
					return err
				}
				value = v
			}

			statID := int32(j)
			if instance.Stats[statID] == nil {
				instance.Stats[statID] = []StatValue{}
			}
			instance.Stats[statID] = append(instance.Stats[statID], StatValue{
				Timestamp: time.Unix(0, gp.currentTime*int64(time.Millisecond)),
				Value:     value,
			})
		}
	}

	return nil
}

// Helper methods for reading Geode format data

func (gp *GeodeParser) readUTF() (string, error) {
	// Read length (2 bytes)
	var length uint16
	if err := binary.Read(gp.reader, gp.byteOrder, &length); err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	// Read UTF-8 bytes
	bytes := make([]byte, length)
	if _, err := io.ReadFull(gp.reader, bytes); err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (gp *GeodeParser) readResourceID() (int, error) {
	b, err := gp.reader.ReadByte()
	if err != nil {
		return 0, err
	}

	if b < SHORT_RESOURCE_INST_ID_TOKEN {
		return int(b), nil
	}

	switch b {
	case SHORT_RESOURCE_INST_ID_TOKEN:
		var id uint16
		if err := binary.Read(gp.reader, gp.byteOrder, &id); err != nil {
			return 0, err
		}
		return int(id), nil
	case INT_RESOURCE_INST_ID_TOKEN:
		var id uint32
		if err := binary.Read(gp.reader, gp.byteOrder, &id); err != nil {
			return 0, err
		}
		return int(id), nil
	default:
		return 0, fmt.Errorf("invalid resource ID token: %d", b)
	}
}

func (gp *GeodeParser) readCompactValue() (int64, error) {
	b, err := gp.reader.ReadByte()
	if err != nil {
		return 0, err
	}

	if b >= 0 {
		return int64(int8(b)), nil
	}

	// Handle compact value tokens
	switch int8(b) {
	case COMPACT_VALUE_2_TOKEN:
		var val int16
		if err := binary.Read(gp.reader, gp.byteOrder, &val); err != nil {
			return 0, err
		}
		return int64(val), nil
	case COMPACT_VALUE_4_TOKEN:
		var val int32
		if err := binary.Read(gp.reader, gp.byteOrder, &val); err != nil {
			return 0, err
		}
		return int64(val), nil
	case COMPACT_VALUE_8_TOKEN:
		var val int64
		if err := binary.Read(gp.reader, gp.byteOrder, &val); err != nil {
			return 0, err
		}
		return val, nil
	default:
		return 0, fmt.Errorf("unsupported compact value token: %d", b)
	}
}

func (gp *GeodeParser) readInt() (int, error) {
	var val int32
	err := binary.Read(gp.reader, gp.byteOrder, &val)
	return int(val), err
}

func (gp *GeodeParser) readInt32() (int32, error) {
	var val int32
	err := binary.Read(gp.reader, gp.byteOrder, &val)
	return val, err
}

func (gp *GeodeParser) readLong() (int64, error) {
	var val int64
	err := binary.Read(gp.reader, gp.byteOrder, &val)
	return val, err
}

func (gp *GeodeParser) readShort() (int16, error) {
	var val int16
	err := binary.Read(gp.reader, gp.byteOrder, &val)
	return val, err
}

func (gp *GeodeParser) readDouble() (float64, error) {
	var val float64
	err := binary.Read(gp.reader, gp.byteOrder, &val)
	return val, err
}

// ParseGeode is the main parsing method that uses the Geode format
func (p *Parser) ParseGeode() error {
	// Create a Geode parser instance with fresh reader
	gp := &GeodeParser{
		file:          p.file,
		reader:        bufio.NewReader(p.file),
		byteOrder:     binary.LittleEndian,
		resourceTypes: make(map[int]*ResourceType),
		instances:     make(map[int]*ResourceInstance),
	}

	// Parse header to set up state
	if err := gp.parseHeader(); err != nil {
		return fmt.Errorf("failed to parse header: %w", err)
	}

	// Set current time
	gp.currentTime = gp.startTime
	p.baseTime = time.Unix(0, gp.startTime*int64(time.Millisecond))

	// Parse records
	if err := gp.parseRecords(p); err != nil {
		return fmt.Errorf("failed to parse records: %w", err)
	}

	return nil
}