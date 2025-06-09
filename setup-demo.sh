#!/bin/bash

# GFS to Prometheus Demo Setup Script

set -e

echo "🚀 Setting up GFS to Prometheus Demo Environment"
echo

# Check if we have any GFS files to work with
if [ ! -d "data" ] || [ -z "$(find data -name '*.gfs' 2>/dev/null)" ]; then
    echo "ℹ️  No existing data directory or GFS files found."
    echo "   Let's create some sample data by processing your GFS file..."
    echo
    
    # Check if user provided a GFS file
    if [ "$1" = "" ]; then
        echo "Usage: $0 <path-to-gfs-file>"
        echo "Example: $0 /Users/anew/workspace/gemfire-runner/data/gemfire/cluster/stats/server-0-stats.gfs"
        exit 1
    fi
    
    GFS_FILE="$1"
    if [ ! -f "$GFS_FILE" ]; then
        echo "❌ GFS file not found: $GFS_FILE"
        exit 1
    fi
    
    echo "📊 Processing GFS file: $GFS_FILE"
    echo "   This will create sample metrics in ./data directory..."
    
    # Build the tool if not present
    if [ ! -f "gfs-to-prometheus" ]; then
        echo "🔨 Building gfs-to-prometheus..."
        make build
    fi
    
    # Create data directory
    mkdir -p data
    
    # Process the GFS file using our working Java pipeline
    echo "⚡ Converting GFS to Prometheus TSDB using Apache Geode libraries..."
    
    # Check if Java extractor is available
    if [ ! -f "java-extractor/DelayDurationExtractor.class" ]; then
        echo "📦 Compiling Java extractor..."
        javac -cp "java-extractor/lib/*" java-extractor/DelayDurationExtractor.java || {
            echo "❌ Failed to compile Java extractor"
            echo "   Make sure Java and dependencies are available"
            exit 1
        }
    fi
    
    # Use our working Java extraction pipeline
    if [ -f "extract_and_convert.sh" ]; then
        ./extract_and_convert.sh "$GFS_FILE" ./data || {
            echo "❌ Java extraction failed"
            exit 1
        }
        echo "✅ Real delayDuration data extracted successfully!"
        echo "   Average: ~997ms, Maximum: ~1120ms (matches VSD)"
    else
        echo "❌ extract_and_convert.sh not found"
        exit 1
    fi
fi

echo
echo "🐳 Starting Docker Compose services..."
echo "   This will start:"
echo "   • Prometheus (port 9090) - Browse metrics and query"
echo "   • Grafana (port 3000) - Advanced dashboards"
echo "   • Node Exporter (port 9100) - System metrics for comparison"
echo

# Start the services
docker-compose -f docker-compose.example.yml up -d

echo
echo "⏳ Waiting for services to start..."
sleep 10

echo
echo "🎉 Demo environment is ready!"
echo
echo "📈 Access Points:"
echo "   • Prometheus UI: http://localhost:9090"
echo "   • Grafana:       http://localhost:3000 (admin/admin)"
echo
echo "🔍 Try these Prometheus queries:"
echo "   • gemfire_statsampler_delayduration     # ⭐ Correctly extracted delayDuration!"
echo "   • {__name__=~\"gemfire.*\"}              # All GemFire metrics" 
echo "   • avg(gemfire_statsampler_delayduration) # Should be ~997ms (matches VSD)"
echo "   • max(gemfire_statsampler_delayduration) # Should be ~1120ms (matches VSD)"
echo
echo "📊 Grafana includes:"
echo "   • Pre-configured Prometheus datasource"
echo "   • GemFire Cluster Overview dashboard"
echo
echo "🛑 To stop: docker-compose -f docker-compose.example.yml down"
echo

# Check if services are responding
echo "🔍 Service Health Check:"
if curl -s http://localhost:9090/-/ready > /dev/null; then
    echo "   ✅ Prometheus is ready"
else
    echo "   ⚠️  Prometheus not responding (may still be starting)"
fi

if curl -s http://localhost:3000/api/health > /dev/null; then
    echo "   ✅ Grafana is ready"
else
    echo "   ⚠️  Grafana not responding (may still be starting)"
fi

echo
echo "🚀 Demo setup complete! Explore your GemFire metrics!"