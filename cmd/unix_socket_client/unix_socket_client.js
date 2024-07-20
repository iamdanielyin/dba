const net = require('net');

// 定义对象结构
class Message {
    constructor(text, id) {
        this.text = text;
        this.id = id;
    }
}

const socketPath = '/tmp/unix_socket';
const client = net.createConnection(socketPath);

client.on('connect', () => {
    // 发送对象
    const msg = new Message('Hello, server!', 1);
    client.write(JSON.stringify(msg));
});

client.on('data', (data) => {
    const resp = JSON.parse(data);
    console.log('Received response:', resp);
    client.end();
});

client.on('error', (err) => {
    console.error('Connection error:', err);
});

client.on('end', () => {
    console.log('Disconnected from server');
});
