import java.io.*;
import org.apache.geode.internal.statistics.StatArchiveReader;

public class TestGeodeApi {
    public static void main(String[] args) {
        try {
            // Test with a dummy file to see available methods
            System.out.println("Testing Geode API methods...");
            
            // This will help us understand the actual API
            Class<?> readerClass = StatArchiveReader.class;
            System.out.println("StatArchiveReader methods:");
            for (java.lang.reflect.Method method : readerClass.getMethods()) {
                if (method.getDeclaringClass() == readerClass) {
                    System.out.println("  " + method.getName() + "()");
                }
            }
            
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}