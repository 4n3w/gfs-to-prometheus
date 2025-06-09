import java.io.*;
import java.util.*;
import org.apache.geode.internal.statistics.StatArchiveReader;

public class TestResourceInst {
    public static void main(String[] args) {
        try {
            Class<?> resourceInstClass = Class.forName("org.apache.geode.internal.statistics.StatArchiveReader$ResourceInst");
            System.out.println("ResourceInst methods:");
            for (java.lang.reflect.Method method : resourceInstClass.getMethods()) {
                if (method.getDeclaringClass() == resourceInstClass) {
                    System.out.println("  " + method.getName() + "()");
                }
            }
            
            // Also check ResourceType
            Class<?> resourceTypeClass = Class.forName("org.apache.geode.internal.statistics.StatArchiveReader$ResourceType");
            System.out.println("\nResourceType methods:");
            for (java.lang.reflect.Method method : resourceTypeClass.getMethods()) {
                if (method.getDeclaringClass() == resourceTypeClass) {
                    System.out.println("  " + method.getName() + "()");
                }
            }
            
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}