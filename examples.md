# GFS to Prometheus Usage Examples

## Docker Compose Setups

### Standard Docker Compose Structure
```bash
# Directory structure:
# docker-compose/
# ├── server-1/
# │   └── stats/server-1-stats.gfs
# ├── server-2/
# │   └── stats/server-2-stats.gfs
# └── locator-1/
#     └── stats/locator-1-stats.gfs

./gfs-to-prometheus cluster docker-compose/ \
  --cluster-name production \
  --node-pattern "*/stats/*-stats.gfs"
```

### Volume-Mounted Data
```bash
# Directory structure:
# volumes/
# ├── gemfire-server-1/
# │   └── data/server-1-stats.gfs
# ├── gemfire-server-2/
# │   └── data/server-2-stats.gfs
# └── gemfire-locator/
#     └── data/locator-stats.gfs

./gfs-to-prometheus cluster volumes/ \
  --cluster-name production \
  --node-pattern "*/data/*-stats.gfs" \
  --node-pattern "*/*/*-stats.gfs"
```

### Kubernetes Persistent Volumes
```bash
# Directory structure:
# k8s-data/
# ├── gemfire-server-0/
# │   └── persistent-data/stats.gfs
# ├── gemfire-server-1/
# │   └── persistent-data/stats.gfs
# └── gemfire-locator-0/
#     └── persistent-data/stats.gfs

./gfs-to-prometheus cluster k8s-data/ \
  --cluster-name k8s-prod \
  --node-pattern "*/persistent-data/*.gfs"
```

### Custom Patterns
```bash
# For complex setups with custom naming
./gfs-to-prometheus cluster /var/gemfire/ \
  --cluster-name custom \
  --node-pattern "*/logs/*.gfs" \
  --node-pattern "*/data/*.gfs" \
  --node-pattern "stats/*/*.gfs" \
  --exclude "*/tmp/*" \
  --exclude "*/backup/*"
```

## Watch Mode Examples

### Real-time Processing
```bash
# Watch for new files in multiple directories
./gfs-to-prometheus cluster-watch \
  /var/docker-volumes/gemfire/ \
  /opt/k8s-data/gemfire/ \
  --cluster-name hybrid \
  --concurrency 8
```

### Development Environment
```bash
# Watch local development cluster
./gfs-to-prometheus cluster-watch ./local-cluster/ \
  --cluster-name dev \
  --node-pattern "*/stats/*.gfs" \
  --tsdb-path ./dev-metrics
```

## Configuration Examples

### Config File (config.yaml)
```yaml
metric_prefix: gemfire

# Cluster-specific filtering
filters:
  include_resource_types:
    - CachePerfStats
    - DistributionStats
    - VMStats
  exclude_stats:
    - debugMetrics

# Custom metric mappings with cluster awareness
metric_mappings:
  "CachePerfStats.puts":
    name: gemfire_cache_operations_total
    labels:
      operation: put
```

## Resulting Metrics

The cluster mode produces metrics with rich labels:

```
# Single node metrics
gemfire_cacheperfstats_gets{cluster="production",node="server-1",node_type="server",resource_type="CachePerfStats",instance="cache"} 12345

# Locator metrics  
gemfire_distributionstats_sentmessages{cluster="production",node="locator-1",node_type="locator",resource_type="DistributionStats",instance="distribution"} 67890

# Environment inference
gemfire_vmstats_heapused{cluster="production-cluster",node="server-2",node_type="server",environment="production",resource_type="VMStats",instance="vm"} 1024000
```

## Grafana Dashboard Queries

```promql
# Cluster-wide cache operations
sum(rate(gemfire_cacheperfstats_gets[5m])) by (cluster)

# Per-node memory usage
gemfire_vmstats_heapused{cluster="production"} by (node, node_type)

# Locator vs Server comparison
sum(rate(gemfire_distributionstats_sentmessages[5m])) by (node_type)
```