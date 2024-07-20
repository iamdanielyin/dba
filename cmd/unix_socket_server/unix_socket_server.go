package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

func main() {
	socketPath := "/tmp/unix_socket"

	// 确保没有旧的socket文件
	if err := os.RemoveAll(socketPath); err != nil {
		fmt.Println("Error removing old socket:", err)
		return
	}

	// 监听Unix Socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("Error setting up Unix socket:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server is listening...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

// 定义对象结构
type Message struct {
	Text string `json:"text"`
	ID   int    `json:"id"`
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	var msg Message

	// 读取数据
	decoder := json.NewDecoder(conn)
	err := decoder.Decode(&msg)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	fmt.Printf("Received: %+v\n", msg)

	// 回应
	msg.Text = "Acknowledged: " + msg.Text
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(&msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
	}
}
