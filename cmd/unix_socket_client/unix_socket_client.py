import json
import socket
import os

# 定义对象结构
class Message:
    def __init__(self, text, msg_id):
        self.text = text
        self.id = msg_id

def main():
    socket_path = "/tmp/unix_socket"

    # 创建Unix Socket
    client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    client.connect(socket_path)

    # 发送对象
    msg = Message("Hello, server!", 1)
    msg_json = json.dumps(msg.__dict__)
    client.sendall(msg_json.encode('utf-8'))

    # 接收回应
    data = client.recv(1024)
    resp_json = data.decode('utf-8')
    resp = json.loads(resp_json)

    print(f"Received response: {resp}")

    client.close()

if __name__ == "__main__":
    main()
