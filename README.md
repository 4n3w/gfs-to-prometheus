# GFS to Prometheus Converter

Convert GemFire/Geode statistics files (.gfs) to Prometheus TSDB format for modern monitoring and analysis.

## Features

- Parse binary GFS statistics files
- Direct write to Prometheus TSDB (no remote write overhead)
- Batch conversion of existing files
- Real-time monitoring with file watcher
- Configurable metric naming and filtering

## Installation

```bash
go mod download
go build -o gfs-to-prometheus
```

## Usage

### Batch Conversion

Convert existing GFS files:

```bash
# Single file
./gfs-to-prometheus convert stats.gfs

# Multiple files
./gfs-to-prometheus convert *.gfs

# With custom TSDB path
./gfs-to-prometheus --tsdb-path /path/to/prometheus/data convert stats.gfs
```

### Watch Mode

Monitor directories for new GFS files:

```bash
# Watch current directory
./gfs-to-prometheus watch

# Watch specific directories
./gfs-to-prometheus watch --dir /path/to/gfs/files --dir /another/path

# With config file
./gfs-to-prometheus --config config.yaml watch --dir /path/to/gfs
```

## Configuration

Create a `config.yaml` file to customize metric conversion:

```yaml
metric_prefix: gemfire

filters:
  include_resource_types:
    - CachePerfStats
    - PartitionedRegionStats
  exclude_stats:
    - internalStats

metric_mappings:
  "CachePerfStats.puts":
    name: cache_operations_total
    labels:
      operation: put
```

## Metric Format

GFS statistics are converted to Prometheus metrics with the following format:

```
<prefix>_<resource_type>_<stat_name>{resource_type="...", instance="..."}
```

Example:
```
gemfire_cacheperfstats_puts{resource_type="CachePerfStats", instance="server1"} 12345
```