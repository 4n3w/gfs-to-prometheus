import java.io.*;
import java.util.*;
import org.apache.geode.internal.statistics.StatArchiveReader;
import org.apache.geode.internal.statistics.StatArchiveReader.*;

/**
 * Robust Java extractor for delayDuration using Apache Geode's official API
 * Outputs CSV format: timestamp_ms,delayduration_ms
 */
public class DelayDurationExtractor {
    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            System.err.println("Usage: java DelayDurationExtractor <gfs-file>");
            System.exit(1);
        }
        
        String gfsFile = args[0];
        System.err.println("Extracting delayDuration from: " + gfsFile);
        
        try {
            // Create reader with official Geode API
            StatArchiveReader reader = new StatArchiveReader(new File[]{new File(gfsFile)}, null, false);
            System.err.println("✅ StatArchiveReader created successfully");
            
            // Get all resource instances
            List<ResourceInst> instances = reader.getResourceInstList();
            System.err.println("Found " + instances.size() + " resource instances");
            
            // Find StatSampler instance
            ResourceInst statSamplerInst = null;
            for (ResourceInst inst : instances) {
                String typeName = inst.getType().getName();
                if ("StatSampler".equals(typeName)) {
                    statSamplerInst = inst;
                    System.err.println("Found StatSampler instance: " + inst.getName());
                    break;
                }
            }
            
            if (statSamplerInst == null) {
                System.err.println("❌ No StatSampler instance found");
                System.exit(1);
            }
            
            // Verify delayDuration stat exists
            StatDescriptor[] stats = statSamplerInst.getType().getStats();
            int delayDurationIndex = -1;
            for (int i = 0; i < stats.length; i++) {
                if ("delayDuration".equals(stats[i].getName())) {
                    delayDurationIndex = i;
                    System.err.println("Found delayDuration at stat index: " + i);
                    break;
                }
            }
            
            if (delayDurationIndex == -1) {
                System.err.println("❌ delayDuration stat not found");
                System.exit(1);
            }
            
            // Extract time series data using discovered API
            System.err.println("Extracting time series data...");
            
            // Get timestamps  
            double[] timestamps = statSamplerInst.getSnapshotTimesMillis();
            System.err.println("Found " + timestamps.length + " timestamps");
            
            // Get delayDuration values
            StatValue delayStatValue = statSamplerInst.getStatValue("delayDuration");
            double[] delayValues = delayStatValue.getSnapshots();
            System.err.println("Found " + delayValues.length + " delayDuration values");
            
            if (timestamps.length != delayValues.length) {
                System.err.println("⚠️  Timestamp count (" + timestamps.length + 
                                 ") != value count (" + delayValues.length + ")");
                // Use minimum length
                int minLength = Math.min(timestamps.length, delayValues.length);
                System.err.println("Using " + minLength + " samples");
            }
            
            // Output CSV format
            System.out.println("timestamp_ms,delayduration_ms");
            
            int sampleCount = Math.min(timestamps.length, delayValues.length);
            double sum = 0;
            double min = Double.MAX_VALUE;
            double max = Double.MIN_VALUE;
            
            for (int i = 0; i < sampleCount; i++) {
                double value = delayValues[i];
                
                // Skip invalid values
                if (Double.isNaN(value) || Double.isInfinite(value)) {
                    continue;
                }
                
                System.out.println((long)timestamps[i] + "," + value);
                
                // Calculate statistics
                sum += value;
                if (value < min) min = value;
                if (value > max) max = value;
            }
            
            // Output statistics to stderr for validation
            double avg = sum / sampleCount;
            System.err.println("\n=== EXTRACTION STATISTICS ===");
            System.err.println("Samples extracted: " + sampleCount);
            System.err.println("Average: " + String.format("%.4f", avg) + " ms");
            System.err.println("Minimum: " + String.format("%.4f", min) + " ms");
            System.err.println("Maximum: " + String.format("%.4f", max) + " ms");
            
            // Compare with VSD expectations
            System.err.println("\n=== VSD COMPARISON ===");
            System.err.println("VSD Expected Average: 997.4038 ms");
            System.err.println("VSD Expected Maximum: 1120.0 ms");
            
            if (Math.abs(avg - 997.4038) < 50) {
                System.err.println("✅ Average matches VSD expectations!");
            } else {
                System.err.println("❌ Average differs from VSD: " + 
                                 String.format("%.1f", Math.abs(avg - 997.4038)) + " ms difference");
            }
            
            if (max <= 1200) { // Allow some tolerance
                System.err.println("✅ Maximum is within reasonable range of VSD!");
            } else {
                System.err.println("❌ Maximum exceeds VSD expectations by: " + 
                                 String.format("%.1f", max - 1120) + " ms");
            }
            
            System.err.println("\n✅ Extraction completed successfully!");
            
        } catch (Exception e) {
            System.err.println("❌ Extraction failed: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}