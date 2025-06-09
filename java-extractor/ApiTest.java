import java.io.*;
import java.util.*;
import java.lang.reflect.*;
import org.apache.geode.internal.statistics.StatArchiveReader;
import org.apache.geode.internal.statistics.StatArchiveReader.*;

/**
 * Test program to discover the correct Apache Geode API for extracting statistics
 */
public class ApiTest {
    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            System.err.println("Usage: java ApiTest <gfs-file>");
            System.exit(1);
        }
        
        String gfsFile = args[0];
        System.err.println("Testing Geode API with file: " + gfsFile);
        
        try {
            // Create reader
            StatArchiveReader reader = new StatArchiveReader(new File[]{new File(gfsFile)}, null, false);
            System.err.println("✅ StatArchiveReader created successfully");
            
            // Test resource list access
            List<ResourceInst> instances = reader.getResourceInstList();
            System.err.println("✅ Found " + instances.size() + " resource instances");
            
            // Find StatSampler
            for (ResourceInst inst : instances) {
                String typeName = inst.getType().getName();
                if ("StatSampler".equals(typeName)) {
                    System.err.println("\n=== StatSampler Instance: " + inst.getName() + " ===");
                    
                    // Get stats definition
                    StatDescriptor[] stats = inst.getType().getStats();
                    for (int i = 0; i < stats.length; i++) {
                        StatDescriptor stat = stats[i];
                        if ("delayDuration".equals(stat.getName())) {
                            System.err.println("Found delayDuration stat at index " + i);
                            
                            // Try different approaches to get data
                            System.err.println("Testing data access approaches...");
                            
                            // Approach 1: Direct reflection to find available methods
                            Method[] methods = ResourceInst.class.getMethods();
                            System.err.println("Available methods on ResourceInst:");
                            for (Method m : methods) {
                                if (m.getName().contains("Time") || m.getName().contains("Stat") || 
                                    m.getName().contains("Value") || m.getName().contains("Snapshot")) {
                                    System.err.println("  " + m.getName() + "(" + 
                                        Arrays.toString(m.getParameterTypes()) + ")");
                                }
                            }
                            
                            return; // Found what we need
                        }
                    }
                }
            }
            
        } catch (Exception e) {
            System.err.println("❌ Failed: " + e.getMessage());
            e.printStackTrace();
        }
    }
}