<?php

class Message {
    public $text;
    public $id;

    public function __construct($text, $id) {
        $this->text = $text;
        $this->id = $id;
    }

    public function toJson() {
        return json_encode($this);
    }

    public static function fromJson($jsonStr) {
        $data = json_decode($jsonStr, true);
        return new Message($data['text'], $data['id']);
    }
}

$socketPath = '/tmp/unix_socket';

$client = stream_socket_client("unix://$socketPath", $errno, $errstr);
if (!$client) {
    echo "Error: $errstr ($errno)\n";
    exit(1);
}

// 发送对象
$msg = new Message("Hello, server!", 1);
fwrite($client, $msg->toJson());

// 接收回应
$response = fread($client, 1024);
$respMsg = Message::fromJson($response);

echo "Received response: {$respMsg->text}, {$respMsg->id}\n";

fclose($client);

?>
