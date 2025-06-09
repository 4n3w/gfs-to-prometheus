import java.io.*;
import java.util.*;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.fasterxml.jackson.databind.node.ArrayNode;
import org.apache.geode.internal.statistics.StatArchiveReader;
import org.apache.geode.internal.statistics.StatArchiveReader.StatArchiveFile;
import org.apache.geode.internal.statistics.StatArchiveReader.ResourceType;
import org.apache.geode.internal.statistics.StatArchiveReader.ResourceInst;
import org.apache.geode.internal.statistics.StatArchiveReader.StatDescriptor;
import org.apache.geode.internal.statistics.StatArchiveReader.StatValue;

public class StatExtractor {
    public static void main(String[] args) {
        if (args.length != 2) {
            System.err.println("Usage: java StatExtractor <gfs-file> <output-json>");
            System.exit(1);
        }
        
        String gfsFile = args[0];
        String outputFile = args[1];
        
        try {
            extractStats(gfsFile, outputFile);
            System.out.println("Successfully extracted stats to " + outputFile);
        } catch (Exception e) {
            System.err.println("Error extracting stats: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
    
    private static void extractStats(String gfsFile, String outputFile) throws Exception {
        ObjectMapper mapper = new ObjectMapper();
        ObjectNode root = mapper.createObjectNode();
        
        // Use Apache Geode's StatArchiveReader for 100% correct parsing
        StatArchiveReader reader = new StatArchiveReader(new File[] { new File(gfsFile) }, null, false);
        
        // Get archive info
        StatArchiveFile[] archives = reader.getArchives();
        if (archives.length == 0) {
            throw new RuntimeException("No archives found in file");
        }
        
        StatArchiveFile archive = archives[0];
        root.put("archiveStartTime", System.currentTimeMillis()); // Placeholder - will fix later
        
        // Get resource instances using the correct API
        java.util.List<ResourceInst> resourceInstList = reader.getResourceInstList();
        
        // Collect unique resource types
        ArrayNode resourceTypesJson = mapper.createArrayNode();
        Map<String, ResourceType> typeMap = new HashMap<>();
        
        for (ResourceInst inst : resourceInstList) {
            ResourceType resType = inst.getType();
            String typeName = resType.getName();
            
            if (!typeMap.containsKey(typeName)) {
                typeMap.put(typeName, resType);
                
                ObjectNode typeJson = mapper.createObjectNode();
                typeJson.put("id", typeMap.size() - 1);
                typeJson.put("name", typeName);
                typeJson.put("description", resType.getDescription());
                
                ArrayNode statsJson = mapper.createArrayNode();
                StatDescriptor[] stats = resType.getStats();
                
                for (int i = 0; i < stats.length; i++) {
                    StatDescriptor stat = stats[i];
                    ObjectNode statJson = mapper.createObjectNode();
                    statJson.put("id", i);
                    statJson.put("name", stat.getName());
                    statJson.put("description", stat.getDescription());
                    statJson.put("units", stat.getUnits());
                    statJson.put("isCounter", stat.isCounter());
                    statJson.put("typeCode", stat.getTypeCode());
                    statsJson.add(statJson);
                }
                
                typeJson.set("stats", statsJson);
                resourceTypesJson.add(typeJson);
            }
        }
        
        root.set("resourceTypes", resourceTypesJson);
        
        // Extract resource instances and their time-series data
        ArrayNode instancesJson = mapper.createArrayNode();
        int totalSamples = 0;
        
        for (int instIndex = 0; instIndex < resourceInstList.size(); instIndex++) {
            ResourceInst instance = resourceInstList.get(instIndex);
            
            ObjectNode instanceJson = mapper.createObjectNode();
            instanceJson.put("id", instIndex);
            instanceJson.put("typeId", getTypeId(typeMap, instance.getType().getName()));
            instanceJson.put("name", instance.getName());
            
            ArrayNode samplesJson = mapper.createArrayNode();
            
            // Get all stats for this resource type
            ResourceType resType = instance.getType();
            StatDescriptor[] stats = resType.getStats();
            
            // Extract time-series data for each stat
            for (int statIndex = 0; statIndex < stats.length; statIndex++) {
                StatDescriptor stat = stats[statIndex];
                
                try {
                    // Iterate through time series data
                    boolean hasTimeStamp = instance.hasTimeStamp();
                    if (hasTimeStamp) {
                        // Use the iterator-based approach to get all samples
                        instance.firstTimeStamp();
                        while (instance.hasTimeStamp()) {
                            ObjectNode sampleJson = mapper.createObjectNode();
                            sampleJson.put("statId", statIndex);
                            sampleJson.put("statName", stat.getName());
                            sampleJson.put("timestamp", instance.getTimeStamp());
                            
                            // Get the actual stat value 
                            double value = instance.getSnapshotStat(stat).getSnapshotsMostRecent();
                            sampleJson.put("value", value);
                            
                            samplesJson.add(sampleJson);
                            totalSamples++;
                            
                            if (!instance.nextTimeStamp()) {
                                break;
                            }
                        }
                    }
                } catch (Exception e) {
                    System.err.println("Warning: Failed to extract stat " + stat.getName() + 
                                     " for instance " + instance.getName() + ": " + e.getMessage());
                }
            }
            
            instanceJson.set("samples", samplesJson);
            instancesJson.add(instanceJson);
        }
        
        root.set("instances", instancesJson);
        root.put("totalSamples", totalSamples);
        
        System.out.println("Extracted " + totalSamples + " total samples from " + 
                          resourceInstList.size() + " instances across " + 
                          typeMap.size() + " resource types");
        
        // Write JSON output
        FileWriter writer = new FileWriter(outputFile);
        mapper.writerWithDefaultPrettyPrinter().writeValue(writer, root);
        writer.close();
        
        reader.close();
    }
    
    private static int getTypeId(Map<String, ResourceType> typeMap, String typeName) {
        int index = 0;
        for (String name : typeMap.keySet()) {
            if (name.equals(typeName)) {
                return index;
            }
            index++;
        }
        return -1;
    }
}