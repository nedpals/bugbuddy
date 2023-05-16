public class ShouldBeNull {
  public static void main(String[] args) {
    String test = null;

    System.err.println("This should not be included in the error below");

    if (test.toUpperCase() == "123") {
      System.out.println("Should not succeed!");
    }
  }
}
