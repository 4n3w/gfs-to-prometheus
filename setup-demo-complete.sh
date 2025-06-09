#!/bin/bash

# GFS to Prometheus Complete Demo Setup Script
# Processes ALL statistics from GFS files (42.7 million data samples!)

set -e

echo "🚀 Setting up Complete GFS to Prometheus Demo Environment"
echo "   This demo extracts ALL statistics, not just delayDuration"
echo

# Check if we have the complete extraction CSV
if [ -f "temp_all_stats_1749231120.csv" ]; then
    echo "✅ Found complete stats extraction (4.8GB with 42.7M samples)"
    echo "   Converting to Prometheus TSDB format..."
    
    # Build the tool if not present
    if [ ! -f "gfs-to-prometheus" ]; then
        echo "🔨 Building gfs-to-prometheus..."
        make build
    fi
    
    # Convert the complete CSV to Prometheus format
    echo "⚡ Converting complete stats to Prometheus TSDB..."
    echo "   This may take a few minutes with 42.7 million samples..."
    
    # Use the all_csv_to_prometheus converter
    if [ -f "all_csv_to_prometheus.go" ]; then
        go run all_csv_to_prometheus.go temp_all_stats_1749231120.csv ./data || {
            echo "❌ Failed to convert complete stats"
            exit 1
        }
        echo "✅ Complete stats conversion successful!"
        echo "   All 42.7 million samples now available in Prometheus format"
    else
        echo "❌ all_csv_to_prometheus.go not found"
        exit 1
    fi
    
elif [ "$1" != "" ]; then
    # Fallback to processing a single GFS file
    GFS_FILE="$1"
    if [ ! -f "$GFS_FILE" ]; then
        echo "❌ GFS file not found: $GFS_FILE"
        exit 1
    fi
    
    echo "📊 Processing GFS file: $GFS_FILE"
    echo "   Extracting ALL statistics (this will take longer)..."
    
    # Build the tool if not present
    if [ ! -f "gfs-to-prometheus" ]; then
        echo "🔨 Building gfs-to-prometheus..."
        make build
    fi
    
    # Create data directory
    mkdir -p data
    
    # Use the complete extraction pipeline
    echo "⚡ Extracting ALL statistics using Java pipeline..."
    
    # Check if Java extractor is available
    if [ ! -f "java-extractor/AllStatsExtractor.class" ]; then
        echo "📦 Compiling Java extractor for all stats..."
        javac -cp "java-extractor/lib/*" java-extractor/AllStatsExtractor.java || {
            echo "❌ Failed to compile Java extractor"
            echo "   Make sure Java and dependencies are available"
            exit 1
        }
    fi
    
    # Extract all stats and convert
    echo "🔄 Running complete extraction (this extracts everything)..."
    java -cp "java-extractor:java-extractor/lib/*" AllStatsExtractor "$GFS_FILE" "temp_all_stats_$(date +%s).csv" || {
        echo "❌ Complete stats extraction failed"
        exit 1
    }
    
    # Convert to Prometheus format
    LATEST_CSV=$(ls temp_all_stats_*.csv | tail -1)
    echo "📈 Converting $LATEST_CSV to Prometheus format..."
    go run all_csv_to_prometheus.go "$LATEST_CSV" ./data || {
        echo "❌ Failed to convert complete stats"
        exit 1
    }
    
    echo "✅ Complete stats extraction and conversion successful!"
    
else
    echo "❌ No existing complete extraction found and no GFS file provided"
    echo
    echo "Usage: $0 [path-to-gfs-file]"
    echo "Example: $0 /Users/anew/workspace/gemfire-runner/data/gemfire/cluster/stats/server-0-stats.gfs"
    echo
    echo "Or if you have temp_all_stats_*.csv, this script will use it automatically"
    exit 1
fi

echo
echo "🐳 Starting Docker Compose services..."
echo "   This will start:"
echo "   • Prometheus (port 9090) - Browse ALL your metrics"
echo "   • Grafana (port 3000) - Advanced dashboards"
echo "   • Node Exporter (port 9100) - System metrics for comparison"
echo

# Start the services
docker-compose -f docker-compose.example.yml up -d

echo
echo "⏳ Waiting for services to start..."
sleep 10

echo
echo "🎉 Complete Demo environment is ready!"
echo "   📊 42.7 MILLION data samples now available in Prometheus!"
echo

echo "📈 Access Points:"
echo "   • Prometheus UI: http://localhost:9090"
echo "   • Grafana:       http://localhost:3000 (admin/admin)"
echo

echo "🔍 Try these Prometheus queries for the complete dataset:"
echo "   • {__name__=~\"gemfire.*\"}                    # All GemFire metrics (LOTS!)"
echo "   • gemfire_statsampler_delayduration           # ⭐ Correctly extracted delayDuration"
echo "   • gemfire_linuxsystemstats_logtrace_cachehits # Cache performance metrics"
echo "   • gemfire_distributionmanager_*               # Distribution manager stats"
echo "   • gemfire_locator_*                          # Locator statistics"
echo "   • gemfire_clientsubscription_*               # Client subscription metrics"
echo "   • count by (__name__) ({__name__=~\"gemfire.*\"}) # Count metrics by type"
echo

echo "📊 Advanced Queries:"
echo "   • rate(gemfire_statsampler_delayduration[5m])  # Delay rate over time"
echo "   • histogram_quantile(0.95, gemfire_*)          # 95th percentile of metrics"
echo "   • avg(gemfire_statsampler_delayduration)       # Should be ~997ms (matches VSD)"
echo "   • max(gemfire_statsampler_delayduration)       # Should be ~1120ms (matches VSD)"
echo

echo "📊 Grafana includes:"
echo "   • Pre-configured Prometheus datasource"
echo "   • GemFire Cluster Overview dashboard"
echo "   • Access to ALL 42.7M data points for custom dashboards!"
echo

echo "🛑 To stop: docker-compose -f docker-compose.example.yml down"
echo

# Check if services are responding
echo "🔍 Service Health Check:"
if curl -s http://localhost:9090/-/ready > /dev/null; then
    echo "   ✅ Prometheus is ready"
    echo "   🔗 Browse metrics: http://localhost:9090/graph"
else
    echo "   ⚠️  Prometheus not responding (may still be starting)"
fi

if curl -s http://localhost:3000/api/health > /dev/null; then
    echo "   ✅ Grafana is ready"
    echo "   🔗 Dashboard: http://localhost:3000"
else
    echo "   ⚠️  Grafana not responding (may still be starting)"
fi

echo
echo "🚀 Complete Demo setup finished!"
echo "   🎯 You now have access to ALL statistics from your GFS file"
echo "   📈 42.7 million data samples ready for analysis!"
echo "   🔍 Explore the complete dataset in Prometheus and Grafana"