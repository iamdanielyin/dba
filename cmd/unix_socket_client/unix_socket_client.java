import java.io.*;
import java.net.Socket;
import java.nio.file.Files;
import java.nio.file.Paths;
import org.json.JSONObject;

class Message {
    public String text;
    public int id;

    public Message(String text, int id) {
        this.text = text;
        this.id = id;
    }

    public String toJson() {
        JSONObject json = new JSONObject();
        json.put("text", this.text);
        json.put("id", this.id);
        return json.toString();
    }

    public static Message fromJson(String jsonStr) {
        JSONObject json = new JSONObject(jsonStr);
        return new Message(json.getString("text"), json.getInt("id"));
    }
}

public class UnixSocketClient {
    public static void main(String[] args) {
        String socketPath = "/tmp/unix_socket";
        try {
            Socket socket = new Socket(socketPath, 0);

            // 发送对象
            Message msg = new Message("Hello, server!", 1);
            OutputStream output = socket.getOutputStream();
            PrintWriter writer = new PrintWriter(new OutputStreamWriter(output), true);
            writer.println(msg.toJson());

            // 接收回应
            BufferedReader reader = new BufferedReader(new InputStreamReader(socket.getInputStream()));
            String response = reader.readLine();
            Message respMsg = Message.fromJson(response);
            System.out.println("Received response: " + respMsg.text + ", " + respMsg.id);

            socket.close();
        } catch (IOException e) {
            e.printStackTrace();
        }
    }
}
