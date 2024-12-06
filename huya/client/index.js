const huya_danmu = require("./client");
const WebSocket = require("ws");

// 从命令行参数获取 roomid 和端口号
const roomid = process.argv[2];
const port = process.argv[3];

if (!roomid || !port) {
    console.log("请提供房间 ID 和端口号");
    process.exit(1);
}

// 创建 WebSocket 服务器
const wss = new WebSocket.Server({port: parseInt(port)});

wss.on('listening', () => {
    console.log(`WebSocket 服务器已启动，地址为 ws://127.0.0.1:${port}`);
});

wss.on('connection', (ws) => {
    console.log('客户端已连接');
});

// 创建 Huya 弹幕客户端实例
const client = new huya_danmu(roomid);

client.on("connect", () => {
    // console.log(`#OK`);
    broadcast(JSON.stringify({event: "connect", message: "#OK"}));
});

client.on("message", (msg) => {
    switch (msg.type) {
        case "chat":
            const chatMessage = JSON.stringify(msg);
            // console.log(chatMessage);
            broadcast(chatMessage);
            break;
        case "gift":
            const giftMessage = JSON.stringify(msg);
            // console.log(giftMessage);
            broadcast(giftMessage);
            break;
        case "online":
            // 可选：将人气信息广播
            // const onlineMessage = JSON.stringify(msg);
            // console.log(onlineMessage);
            // broadcast(onlineMessage);
            break;
    }
});

// client.on("error", (e) => {
//   console.log(e);
//   broadcast(JSON.stringify({ event: "error", message: e.message }));
// });
//
// client.on("close", () => {
//   console.log("close");
//   broadcast(JSON.stringify({ event: "close", message: "Connection closed" }));
// });

client.start();

// 广播函数，将数据发送到所有连接的客户端
function broadcast(data) {
    wss.clients.forEach(function (client) {
        if (client.readyState === WebSocket.OPEN) {
            client.send(data);
        }
    });
}
