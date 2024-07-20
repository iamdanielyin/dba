package main

import (
	"encoding/json"
	"fmt"
	"net"
)

// 定义对象结构
type Message struct {
	Text string `json:"text"`
	ID   int    `json:"id"`
}

func main() {
	socketPath := "/tmp/unix_socket"

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Println("Error connecting to Unix socket:", err)
		return
	}
	defer conn.Close()

	// 发送对象
	msg := Message{
		Text: "Hello, server!",
		ID:   1,
	}

	encoder := json.NewEncoder(conn)
	err = encoder.Encode(&msg)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	// 接收回应
	var resp Message
	decoder := json.NewDecoder(conn)
	err = decoder.Decode(&resp)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	fmt.Printf("Received response: %+v\n", resp)
}
