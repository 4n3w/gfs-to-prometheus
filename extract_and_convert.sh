#!/bin/bash

# Complete GFS to Prometheus pipeline using official Apache Geode libraries
# Usage: ./extract_and_convert.sh <gfs-file> <output-tsdb-dir>

set -e

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <gfs-file> <output-tsdb-dir>"
    echo "Example: $0 /path/to/stats.gfs ./prometheus_data"
    exit 1
fi

GFS_FILE="$1"
OUTPUT_DIR="$2"

echo "🚀 GFS to Prometheus Pipeline"
echo "   Input:  $GFS_FILE"
echo "   Output: $OUTPUT_DIR"
echo

# Step 1: Extract delayDuration using official Geode API
echo "📊 Step 1: Extracting delayDuration with Apache Geode StatArchiveReader..."
TEMP_CSV="temp_delayduration_$(date +%s).csv"

java -cp "java-extractor/lib/*:java-extractor" DelayDurationExtractor "$GFS_FILE" 2>/dev/null > "$TEMP_CSV"

if [ ! -f "$TEMP_CSV" ] || [ ! -s "$TEMP_CSV" ]; then
    echo "❌ Failed to extract data from GFS file"
    exit 1
fi

# Show extraction stats
echo "✅ Extracted $(wc -l < "$TEMP_CSV") lines of data"

# Step 2: Convert CSV to Prometheus TSDB
echo "📈 Step 2: Converting to Prometheus TSDB format..."

go run csv_to_prometheus.go "$TEMP_CSV" "$OUTPUT_DIR"

# Step 3: Verify the data
echo "🔍 Step 3: Verifying extracted data..."

# Show sample data
echo "Sample extracted values:"
head -6 "$TEMP_CSV" | tail -5

# Cleanup
rm -f "$TEMP_CSV"

echo
echo "✅ Pipeline completed successfully!"
echo "   TSDB data written to: $OUTPUT_DIR"
echo "   Metric name: gemfire_statsampler_delayduration"
echo
echo "Next steps:"
echo "1. Update prometheus.yml to point to $OUTPUT_DIR"
echo "2. Start/restart Prometheus"
echo "3. Query: gemfire_statsampler_delayduration"
echo "4. Expected values: avg ~997ms, max ~1120ms"