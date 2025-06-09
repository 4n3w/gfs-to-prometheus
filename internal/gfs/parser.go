package gfs

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	GFSMagicNumber = 0x044d // Actual GemFire stats file magic number
	HeaderSize     = 256
)

type StatType int

const (
	StatTypeInt StatType = iota
	StatTypeLong
	StatTypeDouble
	StatTypeFloat
)

type ResourceType struct {
	ID          int32
	Name        string
	Description string
	Stats       []StatDescriptor
}

type StatDescriptor struct {
	ID          int32
	Name        string
	Description string
	Type        StatType
	Unit        string
	IsCounter   bool
	LargestBit  byte
}

type ResourceInstance struct {
	ID           int32
	TypeID       int32
	Name         string
	CreationTime time.Time
	Stats        map[int32][]StatValue
}

type StatValue struct {
	Timestamp time.Time
	Value     interface{}
}

type Parser struct {
	file       *os.File
	reader     io.Reader
	byteOrder  binary.ByteOrder
	types      map[int32]*ResourceType
	instances  map[int32]*ResourceInstance
	baseTime   time.Time
}

func NewParser(filename string) (*Parser, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	p := &Parser{
		file:      file,
		reader:    file,
		byteOrder: binary.LittleEndian,
		types:     make(map[int32]*ResourceType),
		instances: make(map[int32]*ResourceInstance),
	}

	if err := p.readHeader(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	return p, nil
}

func (p *Parser) Close() error {
	return p.file.Close()
}

func (p *Parser) readHeader() error {
	// Read first two bytes for magic number
	magicBytes := make([]byte, 2)
	if _, err := io.ReadFull(p.reader, magicBytes); err != nil {
		return err
	}

	// Check magic number (0x4d 0x04 in file = 0x044d in big endian)
	magic := binary.BigEndian.Uint16(magicBytes)
	if magic != GFSMagicNumber {
		return fmt.Errorf("invalid GFS magic number: %x", magic)
	}

	// The format appears to be big endian based on the magic number
	p.byteOrder = binary.BigEndian

	// Skip the rest of the header for now - we'll parse it properly later
	header := make([]byte, 100)
	if _, err := io.ReadFull(p.reader, header); err != nil {
		return err
	}

	return nil
}

func (p *Parser) Parse() error {
	for {
		recordType, err := p.readByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch recordType {
		case 0x01: // Resource type definition
			if err := p.readResourceType(); err != nil {
				return err
			}
		case 0x02: // Resource instance
			if err := p.readResourceInstance(); err != nil {
				return err
			}
		case 0x03: // Sample
			if err := p.readSample(); err != nil {
				return err
			}
		case 0x04: // Timestamp
			if err := p.readTimestamp(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown record type: %x", recordType)
		}
	}

	return nil
}

func (p *Parser) readByte() (byte, error) {
	var b byte
	err := binary.Read(p.reader, p.byteOrder, &b)
	return b, err
}

func (p *Parser) readString() (string, error) {
	length, err := p.readInt16()
	if err != nil {
		return "", err
	}

	bytes := make([]byte, length)
	if _, err := io.ReadFull(p.reader, bytes); err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (p *Parser) readInt16() (int16, error) {
	var val int16
	err := binary.Read(p.reader, p.byteOrder, &val)
	return val, err
}

func (p *Parser) readInt32() (int32, error) {
	var val int32
	err := binary.Read(p.reader, p.byteOrder, &val)
	return val, err
}

func (p *Parser) readInt64() (int64, error) {
	var val int64
	err := binary.Read(p.reader, p.byteOrder, &val)
	return val, err
}

func (p *Parser) readFloat64() (float64, error) {
	var val float64
	err := binary.Read(p.reader, p.byteOrder, &val)
	return val, err
}

func (p *Parser) readResourceType() error {
	typeID, err := p.readInt32()
	if err != nil {
		return err
	}

	name, err := p.readString()
	if err != nil {
		return err
	}

	desc, err := p.readString()
	if err != nil {
		return err
	}

	statCount, err := p.readInt16()
	if err != nil {
		return err
	}

	resType := &ResourceType{
		ID:          typeID,
		Name:        name,
		Description: desc,
		Stats:       make([]StatDescriptor, statCount),
	}

	for i := int16(0); i < statCount; i++ {
		stat, err := p.readStatDescriptor()
		if err != nil {
			return err
		}
		resType.Stats[i] = stat
	}

	p.types[typeID] = resType
	return nil
}

func (p *Parser) readStatDescriptor() (StatDescriptor, error) {
	var stat StatDescriptor

	id, err := p.readInt32()
	if err != nil {
		return stat, err
	}
	stat.ID = id

	name, err := p.readString()
	if err != nil {
		return stat, err
	}
	stat.Name = name

	desc, err := p.readString()
	if err != nil {
		return stat, err
	}
	stat.Description = desc

	typeFlag, err := p.readByte()
	if err != nil {
		return stat, err
	}

	stat.Type = StatType(typeFlag & 0x0F)
	stat.IsCounter = (typeFlag & 0x10) != 0

	unit, err := p.readString()
	if err != nil {
		return stat, err
	}
	stat.Unit = unit

	return stat, nil
}

func (p *Parser) readResourceInstance() error {
	instID, err := p.readInt32()
	if err != nil {
		return err
	}

	typeID, err := p.readInt32()
	if err != nil {
		return err
	}

	name, err := p.readString()
	if err != nil {
		return err
	}

	createTime, err := p.readInt64()
	if err != nil {
		return err
	}

	instance := &ResourceInstance{
		ID:           instID,
		TypeID:       typeID,
		Name:         name,
		CreationTime: time.Unix(0, createTime*int64(time.Millisecond)),
		Stats:        make(map[int32][]StatValue),
	}

	p.instances[instID] = instance
	return nil
}

func (p *Parser) readSample() error {
	instID, err := p.readInt32()
	if err != nil {
		return err
	}

	instance, ok := p.instances[instID]
	if !ok {
		return fmt.Errorf("unknown instance ID: %d", instID)
	}

	resType, ok := p.types[instance.TypeID]
	if !ok {
		return fmt.Errorf("unknown type ID: %d", instance.TypeID)
	}

	timestamp := p.baseTime

	statCount, err := p.readInt16()
	if err != nil {
		return err
	}

	for i := int16(0); i < statCount; i++ {
		statID, err := p.readInt32()
		if err != nil {
			return err
		}

		var statDesc *StatDescriptor
		for _, s := range resType.Stats {
			if s.ID == statID {
				statDesc = &s
				break
			}
		}

		if statDesc == nil {
			return fmt.Errorf("unknown stat ID: %d", statID)
		}

		var value interface{}
		switch statDesc.Type {
		case StatTypeInt:
			v, err := p.readInt32()
			if err != nil {
				return err
			}
			value = v
		case StatTypeLong:
			v, err := p.readInt64()
			if err != nil {
				return err
			}
			value = v
		case StatTypeDouble:
			v, err := p.readFloat64()
			if err != nil {
				return err
			}
			value = v
		}

		if instance.Stats[statID] == nil {
			instance.Stats[statID] = []StatValue{}
		}
		instance.Stats[statID] = append(instance.Stats[statID], StatValue{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	return nil
}

func (p *Parser) readTimestamp() error {
	ts, err := p.readInt64()
	if err != nil {
		return err
	}
	p.baseTime = time.Unix(0, ts*int64(time.Millisecond))
	return nil
}

func (p *Parser) GetInstances() map[int32]*ResourceInstance {
	return p.instances
}

func (p *Parser) GetTypes() map[int32]*ResourceType {
	return p.types
}