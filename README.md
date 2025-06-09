# GFS to Prometheus Converter

Convert GemFire/Geode statistics files (.gfs) to Prometheus TSDB format for modern monitoring and analysis. Built with research-backed understanding of the GFS binary format from Apache Geode source code.

## Features

- **Binary GFS parsing** with proper format detection and byte order handling
- **Direct TSDB writes** for maximum performance (no remote write overhead)
- **Cluster-aware processing** for multi-node GemFire deployments
- **Flexible file discovery** supporting Docker Compose, Kubernetes, and traditional setups
- **Rich metric labeling** with cluster, node, and type information
- **Real-time monitoring** with file system watchers
- **Concurrent processing** for large-scale deployments
- **Configurable metric naming** and filtering

## Installation

```bash
go mod download
go build -o gfs-to-prometheus
```

## Usage

### Single File Processing

Convert individual GFS files:

```bash
# Single file
./gfs-to-prometheus convert stats.gfs

# Multiple files with custom TSDB path
./gfs-to-prometheus --tsdb-path /path/to/prometheus/data convert *.gfs
```

### Cluster Processing (Recommended)

Process entire GemFire clusters with automatic node detection:

```bash
# Docker Compose setup
./gfs-to-prometheus cluster /var/docker-volumes/gemfire/ \
  --cluster-name production \
  --node-pattern "*/stats/*-stats.gfs"

# Kubernetes persistent volumes
./gfs-to-prometheus cluster /opt/k8s-data/gemfire/ \
  --cluster-name k8s-prod \
  --node-pattern "*/persistent-data/*.gfs"

# Traditional deployment
./gfs-to-prometheus cluster /opt/gemfire/cluster/ \
  --cluster-name datacenter-1 \
  --concurrency 8
```

### Real-time Monitoring

Watch for new GFS files across cluster nodes:

```bash
# Watch entire cluster directory tree
./gfs-to-prometheus cluster-watch /var/gemfire/ \
  --cluster-name production \
  --recursive

# Watch multiple deployment locations
./gfs-to-prometheus cluster-watch \
  /var/docker-volumes/gemfire/ \
  /opt/k8s-data/gemfire/ \
  --cluster-name hybrid
```

### File Discovery Patterns

The tool automatically discovers GFS files using flexible patterns:

```bash
# Default patterns (covers most deployments)
*/stats/*-stats.gfs           # server-1/stats/server-1-stats.gfs
*/*/*-stats.gfs               # volumes/server-1/data/server-1-stats.gfs
*/data/*-stats.gfs            # server-1/data/server-1-stats.gfs
*/persistent-data/*-stats.gfs # k8s persistent volumes

# Custom patterns
./gfs-to-prometheus cluster /custom/path/ \
  --node-pattern "*/logs/*.gfs" \
  --node-pattern "stats/*/*.gfs" \
  --exclude "*/tmp/*" \
  --exclude "*/backup/*"
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

### Single Node Metrics
```
gemfire_cacheperfstats_puts{resource_type="CachePerfStats", instance="cache"} 12345
```

### Cluster Metrics (Recommended)
```
gemfire_cacheperfstats_puts{
  cluster="production",
  node="server-1", 
  node_type="server",
  resource_type="CachePerfStats",
  instance="cache",
  environment="production"
} 12345

gemfire_distributionstats_sentmessages{
  cluster="production",
  node="locator-1",
  node_type="locator", 
  resource_type="DistributionStats",
  instance="distribution"
} 67890
```

## Grafana Integration

Query examples for cluster-wide dashboards:

```promql
# Cluster-wide cache operations rate
sum(rate(gemfire_cacheperfstats_gets[5m])) by (cluster)

# Memory usage by node type
gemfire_vmstats_heapused{cluster="production"} by (node, node_type)

# Locator vs Server message distribution
sum(rate(gemfire_distributionstats_sentmessages[5m])) by (node_type)

# Top 5 busiest servers
topk(5, rate(gemfire_cacheperfstats_puts[5m]{node_type="server"}))
```

## Deployment Patterns

### Docker Compose
```yaml
# docker-compose.yml
version: '3.8'
services:
  gfs-monitor:
    build: .
    volumes:
      - ./gemfire-data:/data
      - ./prometheus-data:/tsdb
    command: cluster-watch /data --cluster-name docker-prod
```

### Kubernetes CronJob
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: gfs-processor
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: gfs-processor
            image: gfs-to-prometheus:latest
            command: 
            - ./gfs-to-prometheus
            - cluster
            - /data
            - --cluster-name=k8s-prod
            volumeMounts:
            - name: gemfire-data
              mountPath: /data
            - name: prometheus-data  
              mountPath: /tsdb
```

## Performance

- **Direct TSDB writes**: ~10x faster than remote write API
- **Concurrent processing**: Scales with available CPU cores
- **Memory efficient**: Streams large files without loading entirely into memory
- **Cluster processing**: Handles 100+ nodes with configurable concurrency