#!/bin/bash

# Download missing Geode dependencies
echo "Downloading missing Geode dependencies..."

cd java-extractor/lib

# Log4j dependencies that Geode needs
wget -q https://repo1.maven.org/maven2/org/apache/logging/log4j/log4j-api/2.17.1/log4j-api-2.17.1.jar
wget -q https://repo1.maven.org/maven2/org/apache/logging/log4j/log4j-core/2.17.1/log4j-core-2.17.1.jar

# Geode logging 
wget -q https://repo1.maven.org/maven2/org/apache/geode/geode-logging/1.15.1/geode-logging-1.15.1.jar

# Common logging dependencies
wget -q https://repo1.maven.org/maven2/commons-logging/commons-logging/1.2/commons-logging-1.2.jar

echo "Dependencies downloaded!"
ls -la *.jar | wc -l
echo "JAR files available"