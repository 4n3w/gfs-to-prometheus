# Repository Cleanup Plan

## Essential Files to KEEP

### Core Application
- `main.go` - Main entry point
- `go.mod`, `go.sum` - Go dependencies
- `Makefile` - Build system
- `config.example.yaml` - Configuration example

### Command Structure
- `cmd/` directory - All CLI commands (convert, watch, cluster, root)

### Internal Packages
- `internal/config/` - Configuration handling
- `internal/converter/` - Core conversion logic
- `internal/gfs/` - GFS parsing (parser.go, geode_parser.go, statarchive.go, java_extractor.go)
- `internal/tsdb/` - Prometheus TSDB writer
- `internal/watcher/` - File watching functionality
- `internal/cluster/` - Cluster support

### Java Components (currently required)
- `java-extractor/` directory - Java-based extraction
  - `AllStatsExtractor.java` - Extracts all stats
  - `DelayDurationExtractor.java` - Specific metric extraction
  - `lib/` - Required Java dependencies
  - `build.sh`, `download_deps.sh` - Build scripts

### Documentation
- `README.md` - Main documentation
- `README-WORKING-SOLUTION.md` - Working solution notes
- `README-demo.md` - Demo instructions
- `STREAMING-DESIGN.md` - New streaming design
- `examples.md` - Usage examples

### Infrastructure
- `docker-compose.example.yml` - Docker setup
- `prometheus.yml`, `prometheus_final.yml` - Prometheus configs
- `grafana/` directory - Grafana dashboards

### Scripts
- `setup-demo.sh` - Demo setup
- `setup-demo-complete.sh` - Complete demo with all stats
- `extract_and_convert.sh` - Current working pipeline
- `extract_all_stats.sh` - Extract all statistics

## Files to REMOVE

### Debug/Analysis Scripts (60+ files!)
All these `analyze_*.go`, `debug_*.go`, `check_*.go`, `test_*.go`, `find_*.go`, `fix_*.go` files:
- `analyze_binary_patterns.go`
- `analyze_gfs_format.go`
- `analyze_gfs_timestamps.go`
- `debug_compact_parsing.go`
- `debug_converter_output.go`
- `check_actual_metrics.go`
- `test_combined_types.go`
- etc. (dozens more)

### Test Data Directories
- `test_data/`
- `test_data_fixed/`
- `test_data_improved/`
- `test_data_minimal/`
- `test_data_pipeline/`
- `data_all_stats/`
- `data_correct/`
- `data_final/`
- `output/`
- `output_v2/`
- `output_v3/`

### Temporary Files
- `temp_all_stats_*.csv` - Large CSV files (4.8GB!)
- `delayDuration_*.csv` - Intermediate CSV files
- `*.patch` files

### Standalone JAR files (duplicated in java-extractor/lib/)
- `commons-logging-1.2.jar`
- `geode-logging-1.15.1.jar`
- `log4j-api-2.17.1.jar`
- `log4j-core-2.17.1.jar`

### One-off extractors
- `ExtractDelayDuration.java` (duplicate of java-extractor version)
- `*.class` files in root

### Miscellaneous
- `export_with_gfsh.sh` - Not needed
- `test_prometheus.yml` - Test config

## Recommended .gitignore additions
```
# Test and debug files
test_*.go
debug_*.go
analyze_*.go
check_*.go
find_*.go
fix_*.go

# Test data
test_data*/
data_*/
output*/

# Temporary files
temp_*.csv
*.csv
*.patch

# Java artifacts
*.class
*.jar

# Prometheus data
data/
chunks_head/
wal/
wbl/

# Build artifacts
gfs-to-prometheus
```

## Summary
- **Keep**: ~25 essential files
- **Remove**: ~100+ debug/test files
- **Space saved**: ~5GB+ (mostly from CSV files and test data)