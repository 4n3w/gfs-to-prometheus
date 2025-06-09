import java.io.*;
import java.util.*;
import org.apache.geode.internal.statistics.StatArchiveReader;
import org.apache.geode.internal.statistics.StatArchiveReader.*;

/**
 * Complete Java extractor for ALL GemFire statistics using Apache Geode's official API
 * Outputs CSV format: timestamp_ms,metric_name,resource_type,resource_name,value
 */
public class AllStatsExtractor {
    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            System.err.println("Usage: java AllStatsExtractor <gfs-file>");
            System.exit(1);
        }
        
        String gfsFile = args[0];
        System.err.println("Extracting ALL statistics from: " + gfsFile);
        
        try {
            // Create reader with official Geode API
            StatArchiveReader reader = new StatArchiveReader(new File[]{new File(gfsFile)}, null, false);
            System.err.println("✅ StatArchiveReader created successfully");
            
            // Get all resource instances
            List<ResourceInst> instances = reader.getResourceInstList();
            System.err.println("Found " + instances.size() + " resource instances");
            
            // Output CSV header
            System.out.println("timestamp_ms,metric_name,resource_type,resource_name,value");
            
            int totalSamples = 0;
            int totalMetrics = 0;
            Set<String> uniqueMetrics = new HashSet<>();
            
            // Process each resource instance
            for (ResourceInst inst : instances) {
                String resourceType = inst.getType().getName();
                String resourceName = inst.getName();
                
                System.err.println("Processing: " + resourceType + "." + resourceName);
                
                // Get all stats for this resource type
                StatDescriptor[] stats = inst.getType().getStats();
                
                // Get timestamps for this instance
                double[] timestamps = null;
                try {
                    timestamps = inst.getSnapshotTimesMillis();
                } catch (Exception e) {
                    System.err.println("  ⚠️  No timestamp data for " + resourceName + ": " + e.getMessage());
                    continue;
                }
                
                if (timestamps.length == 0) {
                    System.err.println("  ⚠️  No timestamps for " + resourceName);
                    continue;
                }
                
                // Process each stat
                for (StatDescriptor stat : stats) {
                    String statName = stat.getName();
                    String metricName = formatMetricName(resourceType, statName);
                    uniqueMetrics.add(metricName);
                    
                    try {
                        // Get stat values
                        StatValue statValue = inst.getStatValue(statName);
                        double[] values = statValue.getSnapshots();
                        
                        if (values.length == 0) {
                            continue; // No data for this stat
                        }
                        
                        // Use minimum length to handle timestamp/value mismatches
                        int sampleCount = Math.min(timestamps.length, values.length);
                        
                        // Output all timestamp,value pairs for this stat
                        for (int i = 0; i < sampleCount; i++) {
                            double value = values[i];
                            
                            // Skip invalid values
                            if (Double.isNaN(value) || Double.isInfinite(value)) {
                                continue;
                            }
                            
                            // CSV format: timestamp_ms,metric_name,resource_type,resource_name,value
                            System.out.println((long)timestamps[i] + "," + 
                                             metricName + "," + 
                                             resourceType + "," + 
                                             resourceName + "," + 
                                             value);
                            totalSamples++;
                        }
                        
                        totalMetrics++;
                        
                    } catch (Exception e) {
                        System.err.println("  ⚠️  Failed to extract " + statName + ": " + e.getMessage());
                    }
                }
                
                System.err.println("  ✅ Processed " + stats.length + " stats");
            }
            
            // Output extraction statistics
            System.err.println("\n=== EXTRACTION SUMMARY ===");
            System.err.println("Resource instances processed: " + instances.size());
            System.err.println("Unique metrics extracted: " + uniqueMetrics.size());
            System.err.println("Total metric series: " + totalMetrics);
            System.err.println("Total samples: " + totalSamples);
            
            System.err.println("\n=== UNIQUE METRICS ===");
            List<String> sortedMetrics = new ArrayList<>(uniqueMetrics);
            Collections.sort(sortedMetrics);
            for (String metric : sortedMetrics) {
                System.err.println("  " + metric);
            }
            
            System.err.println("\n✅ Complete extraction finished successfully!");
            
        } catch (Exception e) {
            System.err.println("❌ Extraction failed: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
    
    /**
     * Format metric name according to Prometheus conventions
     */
    private static String formatMetricName(String resourceType, String statName) {
        String prefix = "gemfire";
        
        // Convert to lowercase and replace spaces/dashes with underscores
        resourceType = resourceType.toLowerCase()
                                 .replaceAll("\\s+", "_")
                                 .replaceAll("-", "_");
        
        statName = statName.toLowerCase()
                          .replaceAll("\\s+", "_") 
                          .replaceAll("-", "_");
        
        return prefix + "_" + resourceType + "_" + statName;
    }
}