import java.io.*;
import java.util.*;
import java.lang.reflect.*;
import org.apache.geode.internal.statistics.StatArchiveReader;
import org.apache.geode.internal.statistics.StatArchiveReader.*;

/**
 * Test program to discover the correct Apache Geode API for extracting statistics
 */
public class TestGeodeApi2 {
    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            System.err.println("Usage: java TestGeodeApi2 <gfs-file>");
            System.exit(1);
        }
        
        String gfsFile = args[0];
        System.err.println("Testing Geode API with file: " + gfsFile);
        
        try {
            // Create reader
            StatArchiveReader reader = new StatArchiveReader(new File[]{new File(gfsFile)}, null, false);
            System.err.println("✅ StatArchiveReader created successfully");
            
            // Discover available methods
            System.err.println("\n=== StatArchiveReader Methods ===");
            Method[] methods = StatArchiveReader.class.getMethods();
            for (Method method : methods) {
                if (method.getName().startsWith("get") && method.getParameterCount() == 0) {
                    System.err.println("  " + method.getName() + "() -> " + method.getReturnType().getSimpleName());
                }
            }
            
            // Test resource list access
            try {
                List<ResourceInst> instances = reader.getResourceInstList();
                System.err.println("\n✅ Found " + instances.size() + " resource instances");
                
                // Find StatSampler
                for (ResourceInst inst : instances) {
                    String typeName = inst.getType().getName();
                    if ("StatSampler".equals(typeName)) {
                        System.err.println("\n=== StatSampler Instance: " + inst.getName() + " ===");
                        
                        // Discover ResourceInst methods
                        System.err.println("ResourceInst methods:");
                        Method[] instMethods = ResourceInst.class.getMethods();
                        for (Method method : instMethods) {
                            if (method.getParameterCount() <= 1 && !method.getName().startsWith("wait")) {
                                System.err.println("  " + method.getName() + "(" + 
                                    Arrays.toString(method.getParameterTypes()) + ") -> " + 
                                    method.getReturnType().getSimpleName());
                            }
                        }
                        
                        // Test different ways to get time series data
                        System.err.println("\nTesting time series access methods:");
                        
                        // Method 1: Try getSnapshots
                        try {
                            Method getSnapshots = ResourceInst.class.getMethod("getSnapshots");
                            Object result = getSnapshots.invoke(inst);
                            System.err.println("  getSnapshots() -> " + result.getClass().getSimpleName() + 
                                             " (length: " + Array.getLength(result) + ")");
                        } catch (Exception e) {
                            System.err.println("  getSnapshots() -> NOT AVAILABLE: " + e.getMessage());
                        }
                        
                        // Method 2: Try iterator approach
                        try {
                            Method hasTimeStamp = ResourceInst.class.getMethod("hasTimeStamp");
                            Boolean hasData = (Boolean) hasTimeStamp.invoke(inst);
                            System.err.println("  hasTimeStamp() -> " + hasData);
                            
                            if (hasData) {
                                Method firstTimeStamp = ResourceInst.class.getMethod("firstTimeStamp");
                                firstTimeStamp.invoke(inst);
                                System.err.println("  ✅ Iterator approach available");
                                
                                // Get first timestamp
                                Method getTimeStamp = ResourceInst.class.getMethod("getTimeStamp");
                                Long timestamp = (Long) getTimeStamp.invoke(inst);
                                System.err.println("  First timestamp: " + timestamp + " (" + new Date(timestamp) + ")");
                            }
                        } catch (Exception e) {
                            System.err.println("  Iterator approach -> NOT AVAILABLE: " + e.getMessage());
                        }
                        
                        // Method 3: Try direct stat access
                        StatDescriptor[] stats = inst.getType().getStats();
                        for (int i = 0; i < stats.length; i++) {
                            StatDescriptor stat = stats[i];
                            if ("delayDuration".equals(stat.getName())) {
                                System.err.println("  Found delayDuration at index " + i);
                                
                                // Try different ways to get stat values
                                try {
                                    Method getSnapshotStat = ResourceInst.class.getMethod("getSnapshotStat", StatDescriptor.class);
                                    Object statSnapshot = getSnapshotStat.invoke(inst, stat);
                                    System.err.println("    getSnapshotStat() -> " + statSnapshot.getClass().getSimpleName());
                                    
                                    // Explore the returned object
                                    Method[] statMethods = statSnapshot.getClass().getMethods();
                                    for (Method method : statMethods) {
                                        if (method.getName().contains("Snapshot") || method.getName().contains("Value")) {
                                            System.err.println("      " + method.getName() + "() -> " + method.getReturnType().getSimpleName());
                                        }
                                    }
                                } catch (Exception e) {
                                    System.err.println("    getSnapshotStat() -> NOT AVAILABLE: " + e.getMessage());
                                }
                            }
                        }
                        
                        break; // Found StatSampler, no need to continue
                    }
                }
                
            } catch (Exception e) {
                System.err.println("❌ Resource list access failed: " + e.getMessage());
                e.printStackTrace();
            }
            
        } catch (Exception e) {
            System.err.println("❌ Failed to test Geode API: " + e.getMessage());
            e.printStackTrace();
        }
    }
}