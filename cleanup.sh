#!/bin/bash

# Repository Cleanup Script
# Removes debugging artifacts, test files, and temporary data

set -e

echo "🧹 GFS-to-Prometheus Repository Cleanup"
echo "   This will remove ~100+ debug files and ~5GB of test data"
echo

# Confirm with user
read -p "Are you sure you want to clean up? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

echo
echo "📁 Removing debug and analysis scripts..."
rm -f analyze_*.go
rm -f debug_*.go
rm -f check_*.go
rm -f test_*.go
rm -f find_*.go
rm -f fix_*.go
rm -f decode_*.go
rm -f diagnose_*.go
rm -f trace_*.go
rm -f validate_*.go
rm -f verify_*.go
rm -f isolate_*.go
rm -f extract_*.go
rm -f create_*.go
rm -f apply_*.go
rm -f brute_force_*.go
rm -f final_*.go

echo "📁 Removing test data directories..."
rm -rf test_data/
rm -rf test_data_*/
rm -rf data_all_stats/
rm -rf data_correct/
rm -rf data_final/
rm -rf output/
rm -rf output_v*/

echo "📁 Removing temporary and CSV files..."
rm -f temp_*.csv
rm -f delayDuration_*.csv
rm -f *.csv

echo "📁 Removing patch files..."
rm -f *.patch

echo "📁 Removing duplicate JAR files from root..."
rm -f commons-logging-*.jar
rm -f geode-logging-*.jar
rm -f log4j-*.jar

echo "📁 Removing Java class files from root..."
rm -f *.class
rm -f ExtractDelayDuration.java

echo "📁 Removing other one-off scripts..."
rm -f export_with_gfsh.sh
rm -f test_prometheus.yml
rm -f all_csv_to_prometheus.go
rm -f csv_to_prometheus.go

echo "📁 Cleaning up empty directories..."
find . -type d -empty -delete 2>/dev/null || true

echo
echo "✅ Cleanup complete!"
echo

# Show what's left
echo "📊 Repository status after cleanup:"
echo "   Files remaining: $(find . -type f -not -path "./.git/*" -not -path "./java-extractor/lib/*" | wc -l)"
echo "   Size: $(du -sh . | cut -f1)"
echo

echo "🎯 Essential structure preserved:"
echo "   ✓ Core application (main.go, cmd/, internal/)"
echo "   ✓ Java extractor (java-extractor/)"
echo "   ✓ Documentation (README files)"
echo "   ✓ Configuration (docker-compose, prometheus, grafana)"
echo "   ✓ Build files (Makefile, go.mod)"
echo

echo "💡 Next steps:"
echo "   1. Review the changes with: git status"
echo "   2. Add cleaned files to git: git add -A"
echo "   3. Commit the cleanup: git commit -m 'Clean up debugging artifacts and test data'"