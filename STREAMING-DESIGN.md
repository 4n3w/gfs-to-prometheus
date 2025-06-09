# Streaming GFS to Prometheus Design

## Problem
- Current approach creates massive intermediate CSV files (4.8GB for complete stats)
- Memory inefficient and slow
- Watch command won't work with Java extraction pipeline

## Proposed Solutions

### 1. Direct Streaming Pipeline (High Priority)
```go
// Pseudo-code for streaming approach
type StreamingConverter struct {
    reader   *StatArchiveReader  // Geode reader
    writer   *tsdb.Writer        // Prometheus writer
    batchSize int               // Process N samples at a time
}

func (s *StreamingConverter) Stream() error {
    batch := make([]Sample, 0, s.batchSize)
    
    for s.reader.HasNext() {
        sample := s.reader.Next()
        batch = append(batch, sample)
        
        if len(batch) >= s.batchSize {
            s.writer.WriteBatch(batch)
            batch = batch[:0] // Reset
        }
    }
    
    // Write remaining
    if len(batch) > 0 {
        s.writer.WriteBatch(batch)
    }
}
```

### 2. Java-Go Hybrid Streaming
- Use Java's StatArchiveReader in streaming mode
- Pipe output directly to Go process (no CSV)
- Protocol: Binary or line-oriented JSON

### 3. Pure Go Implementation
- Implement GFS binary format reader in Go
- Eliminate Java dependency entirely
- More complex but most efficient

### 4. New CLI Commands

```bash
# Stream convert (no intermediate files)
gfs-to-prometheus stream input.gfs --output ./data

# Continuous watch with streaming
gfs-to-prometheus watch-stream /path/to/stats/dir --output ./data

# Memory-efficient batch processing
gfs-to-prometheus convert --streaming --batch-size 10000 input.gfs
```

### 5. Architecture Changes

```
Current: GFS → Java → CSV → Go → TSDB
New:     GFS → Java/Go Reader → Stream → TSDB Writer
```

### Implementation Priority
1. Investigate Geode StatArchiveReader streaming capabilities
2. Implement Java process that streams to stdout (binary protocol)
3. Go process reads from stdin and writes directly to TSDB
4. Benchmark memory usage and performance
5. Eventually implement pure Go GFS reader

### Benefits
- No intermediate files
- Constant memory usage regardless of GFS size
- Can process infinitely large files
- Enables true "watch" functionality
- Much faster end-to-end processing