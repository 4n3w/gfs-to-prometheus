#!/bin/bash

# Complete GFS to Prometheus pipeline for ALL statistics using official Apache Geode libraries
# Usage: ./extract_all_stats.sh <gfs-file> <output-tsdb-dir>

set -e

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <gfs-file> <output-tsdb-dir>"
    echo "Example: $0 /path/to/stats.gfs ./prometheus_all_data"
    exit 1
fi

GFS_FILE="$1"
OUTPUT_DIR="$2"

echo "🚀 Complete GFS to Prometheus Pipeline (ALL Statistics)"
echo "   Input:  $GFS_FILE"
echo "   Output: $OUTPUT_DIR"
echo

# Check if Java extractor is compiled
if [ ! -f "java-extractor/AllStatsExtractor.class" ]; then
    echo "📦 Compiling Java extractor for all statistics..."
    javac -cp "java-extractor/lib/*" java-extractor/AllStatsExtractor.java || {
        echo "❌ Failed to compile AllStatsExtractor"
        exit 1
    }
fi

# Step 1: Extract ALL statistics using official Geode API
echo "📊 Step 1: Extracting ALL statistics with Apache Geode StatArchiveReader..."
TEMP_CSV="temp_all_stats_$(date +%s).csv"

java -cp "java-extractor/lib/*:java-extractor" AllStatsExtractor "$GFS_FILE" 2>/dev/null > "$TEMP_CSV"

if [ ! -f "$TEMP_CSV" ] || [ ! -s "$TEMP_CSV" ]; then
    echo "❌ Failed to extract data from GFS file"
    exit 1
fi

# Show extraction stats
TOTAL_LINES=$(wc -l < "$TEMP_CSV")
DATA_LINES=$((TOTAL_LINES - 1))  # Subtract header
echo "✅ Extracted $DATA_LINES data samples"

# Step 2: Convert CSV to Prometheus TSDB
echo "📈 Step 2: Converting to Prometheus TSDB format..."

go run all_csv_to_prometheus.go "$TEMP_CSV" "$OUTPUT_DIR"

# Step 3: Show sample of extracted data
echo "🔍 Step 3: Sample of extracted metrics..."

# Show first few lines of different metric types
echo "Sample data from extraction:"
head -10 "$TEMP_CSV" | tail -5

# Cleanup
rm -f "$TEMP_CSV"

echo
echo "✅ Complete pipeline finished successfully!"
echo "   TSDB data written to: $OUTPUT_DIR"
echo "   All GemFire statistics are now available in Prometheus"
echo
echo "📊 Example Prometheus queries:"
echo "   • {__name__=~\"gemfire.*\"}                    # All GemFire metrics"
echo "   • gemfire_statsampler_delayduration           # DelayDuration (validated)"
echo "   • gemfire_cacheperfstats_gets                 # Cache operations"
echo "   • gemfire_distributionmessagesendstats_*      # Distribution stats"
echo "   • gemfire_vmstats_*                           # JVM statistics"
echo
echo "🎯 Pro tip: Use Grafana to create dashboards with these metrics!"