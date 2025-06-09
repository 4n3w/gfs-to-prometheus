#!/bin/bash

set -e

echo "Building Java GFS extractor..."

# Create lib directory for dependencies
mkdir -p lib

# Download required JARs if they don't exist
if [ ! -f "lib/geode-core-1.15.1.jar" ]; then
    echo "Downloading Apache Geode..."
    curl -L -o lib/geode-core-1.15.1.jar "https://repo1.maven.org/maven2/org/apache/geode/geode-core/1.15.1/geode-core-1.15.1.jar"
fi

if [ ! -f "lib/jackson-databind-2.15.2.jar" ]; then
    echo "Downloading Jackson..."
    curl -L -o lib/jackson-databind-2.15.2.jar "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-databind/2.15.2/jackson-databind-2.15.2.jar"
    curl -L -o lib/jackson-core-2.15.2.jar "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-core/2.15.2/jackson-core-2.15.2.jar"
    curl -L -o lib/jackson-annotations-2.15.2.jar "https://repo1.maven.org/maven2/com/fasterxml/jackson/core/jackson-annotations/2.15.2/jackson-annotations-2.15.2.jar"
fi

# Additional Geode dependencies
if [ ! -f "lib/geode-common-1.15.1.jar" ]; then
    echo "Downloading additional Geode dependencies..."
    curl -L -o lib/geode-common-1.15.1.jar "https://repo1.maven.org/maven2/org/apache/geode/geode-common/1.15.1/geode-common-1.15.1.jar"
fi

# Compile Java source
echo "Compiling StatExtractor.java..."
javac -cp "lib/*" StatExtractor.java

# Create executable JAR
echo "Creating executable JAR..."
mkdir -p build
jar cfe build/stat-extractor.jar StatExtractor *.class

echo "Java extractor built successfully!"
echo "Usage: java -cp 'lib/*:build/stat-extractor.jar' StatExtractor <gfs-file> <output-json>"