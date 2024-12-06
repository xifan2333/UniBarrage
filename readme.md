# UniBarrage 文档 📜✨

**UniBarrage** 是一个高性能实时代理和统一弹幕数据转发器，用于支持多平台（如抖音、哔哩哔哩、快手、斗鱼、虎牙）的直播弹幕转发。  
**UniBarrage** is a high-performance real-time proxy and unified barrage data forwarder, supporting live streaming
platforms such as Douyin, Bilibili, Kuaishou, Douyu, and Huya.

---

## 一、API 接口文档 API Documentation 🌐📖

### 1.1 二进制启动参数 Command Line Parameters ⚙️

| 参数名 Parameter | 类型 Type | 默认值 Default | 描述 Description                 |
|---------------|---------|-------------|--------------------------------|
| `-wsHost`     | string  | `127.0.0.1` | WebSocket 服务的主机地址 Host Address |
| `-wsPort`     | int     | `7777`      | WebSocket 服务的端口号 Port          |
| `-apiHost`    | string  | `127.0.0.1` | API 服务的主机地址 Host Address       |
| `-apiPort`    | int     | `8080`      | API 服务的端口号 Port                |
| `-useProxy`   | bool    | `false`     | 是否启用代理服务 Enable Proxy          |
| `-authToken`  | string  | `""`        | Bearer Token (仅 API)           |

### 示例命令 Example Command 🛠️

```bash
./UniBarrage -wsHost 192.168.1.10 -wsPort 9000 -apiHost 192.168.1.10 -apiPort 8081 -useProxy
```

---

### 1.2 接口列表 API Endpoints 📬

#### 欢迎接口 Welcome Endpoint 👋

- **URL**: `/`
- **方法 Method**: `GET`
- **描述 Description**: 返回欢迎信息，验证服务是否启动 Returns a welcome message to verify if the service is running.

**响应示例 Response Example:**

```json
{
  "code": 200,
  "message": "Hello, UniBarrage!",
  "data": null
}
```

---

#### 获取所有服务状态 Get All Services Status 🔄

- **URL**: `/all`
- **方法 Method**: `GET`
- **描述 Description**: 获取所有正在运行的服务状态 Get the status of all running services.

**响应示例 Response Example:**

```json
{
  "code": 200,
  "message": "获取成功 Retrieved successfully",
  "data": [
    {
      "platform": "douyin",
      "rid": "123456"
    },
    {
      "platform": "bilibili",
      "rid": "789012"
    }
  ]
}
```

---

#### 获取指定平台的所有服务 Get Services for Specific Platform 🔍

- **URL**: `/{platform}`
- **方法 Method**: `GET`
- **描述 Description**: 获取指定平台的所有服务 Get all services for a specific platform.

**请求参数 Request Parameters:**

- `platform`（路径参数 Path Parameter）：需要查询的直播平台名称，例如 `douyin`, `bilibili`, `kuaishou`, `douyu`, `huya`.

**响应示例 Response Example:**

```json
{
  "code": 200,
  "message": "获取成功 Retrieved successfully",
  "data": [
    {
      "platform": "douyin",
      "rid": "123456"
    }
  ]
}
```

---

#### 获取单个服务状态 Get Single Service Status 🧐

- **URL**: `/{platform}/{roomId}`
- **方法 Method**: `GET`
- **描述 Description**: 获取指定房间的服务状态 Get the status of a specific room.

**请求参数 Request Parameters:**

- `platform`（路径参数 Path Parameter）：直播平台名称 Platform name.
- `roomId`（路径参数 Path Parameter）：房间 ID Room ID.

**响应示例 Response Example:**

```json
{
  "code": 200,
  "message": "获取成功 Retrieved successfully",
  "data": {
    "platform": "douyin",
    "rid": "123456"
  }
}
```

---

#### 启动服务 Start Service 🚀

- **URL**: `/{platform}`
- **方法 Method**: `POST`
- **描述 Description**: 启动指定平台的直播服务 Start a live service for a specified platform.

**请求体 Request Body Example:**

```json
{
  "rid": "123456",
  "cookie": "可选的登录cookie Optional login cookie"
}
```

**请求参数 Request Parameters:**

- `platform`（路径参数 Path Parameter）：直播平台名称 Platform name.
- `rid`（请求体参数 Body Parameter）：房间 ID Room ID.
- `cookie`（请求体参数 Body Parameter, 可选 Optional）：用于需要登录的服务 Used for services requiring login.

**响应示例 Response Example:**

```json
{
  "code": 201,
  "message": "服务启动成功 Service started successfully",
  "data": {
    "platform": "douyin",
    "rid": "123456"
  }
}
```

---

#### 停止服务 Stop Service 🛑

- **URL**: `/{platform}/{roomId}`
- **方法 Method**: `DELETE`
- **描述 Description**: 停止指定平台房间的服务 Stop the service for a specific room on a platform.

**请求参数 Request Parameters:**

- `platform`（路径参数 Path Parameter）：直播平台名称 Platform name.
- `roomId`（路径参数 Path Parameter）：房间 ID Room ID.

**响应示例 Response Example:**

```json
{
  "code": 200,
  "message": "服务已停止 Service stopped",
  "data": {
    "platform": "douyin",
    "rid": "123456"
  }
}
```

---

### 错误响应示例 Error Response Example ❗

**示例 Example:**

```json
{
  "code": 404,
  "message": "服务未找到 Service not found",
  "data": null
}
```

---

### 错误码 Error Codes 🚨

| 错误码 Code | 描述 Description                |
|----------|-------------------------------|
| `200`    | 请求成功 Request successful       |
| `201`    | 服务创建成功 Service created        |
| `400`    | 请求参数错误 Bad request parameters |
| `404`    | 服务未找到 Service not found       |
| `500`    | 服务器内部错误 Internal server error |

---

## 二、WebSocket 消息结构 WebSocket Message Structure 📡💬

### 2.1 消息结构 Message Structure 🌐

```json
{
  "rid": "房间号 Room ID",
  "platform": "来源平台 Platform",
  "type": "消息类型 Message Type",
  "data": "消息内容数据 Message Data"
}
```

### 字段说明 Field Descriptions 📜

- **`rid`**: 房间号 Room ID.
- **`platform`**: 来源平台 Platform, 包括 Douyin, Bilibili, 等等.
- **`type`**: 消息类型 Message Type:
    - `Chat` 聊天消息 Chat Message 💬
    - `Gift` 礼物消息 Gift Message 🎁
    - `Like` 点赞消息 Like Message 👍
    - `EnterRoom` 进入房间消息 Enter Room 🏠
    - `Subscribe` 订阅消息 Subscribe Message 🛎️
    - `SuperChat` 超级聊天消息 Super Chat 💵
    - `EndLive` 结束直播 End Live 📴

---

### 2.2 消息内容结构 Message Data Structures 🧩

#### Chat 消息 Chat Message 💬

```json
{
  "name": "发送者名称 Sender",
  "avatar": "发送者头像 URL Avatar URL",
  "content": "聊天内容 Content",
  "emoticon": [
    "表情URL Emoticon URLs"
  ],
  "raw": "原始数据 Raw Data"
}
```

#### Gift 消息 Gift Message 🎁

```json
{
  "name": "赠送者名称 Sender",
  "avatar": "赠送者头像 URL Avatar URL",
  "item": "礼物名称 Gift Name",
  "num": "礼物数量 Gift Quantity",
  "price": "礼物单价 Gift Price",
  "giftIcon": "礼物图标 URL Gift Icon URL",
  "raw": "原始数据 Raw Data"
}
```

#### Like 消息 Like Message 👍

```json
{
  "name": "点赞者名称 Liker",
  "avatar": "点赞者头像 URL Avatar URL",
  "count": "点赞次数 Like Count",
  "raw": "原始数据 Raw Data"
}
```

#### EnterRoom 消息 Enter Room Message 🏠

```json
{
  "name": "进入者名称 Participant",
  "avatar": "进入者头像 URL Avatar URL",
  "raw": "原始数据 Raw Data"
}
```

#### Subscribe 消息 Subscribe Message 🛎️

```json
{
  "name": "订阅者名称 Subscriber",
  "avatar": "订阅者头像 URL Avatar URL",
  "item": "订阅项 Subscription Item",
  "num": "订阅次数 Subscription Count",
  "price": "订阅单价 Subscription Price",
  "raw": "原始数据 Raw Data"
}
```

#### SuperChat 消息 Super Chat Message 💵

```json
{
  "name": "发送者名称 Sender",
  "avatar": "发送者头像 URL Avatar URL",
  "content": "超级聊天内容 Content",
  "price": "金额 Price",
  "raw": "原始数据 Raw Data"
}
```

#### EndLive 消息 End Live Message 📴

```json
{
  "raw": "原始数据 Raw Data"
}
```

---

### 2.3 示例消息 Examples 🌟

**Chat 消息 Chat Message Example:**

```json
{
  "rid": "123456",
  "platform": "douyin",
  "type": "Chat",
  "data": {
    "name": "用户A",
    "avatar": "https://example.com/avatar.jpg",
    "content": "这是聊天消息 This is a chat message."
  }
}
```

**Gift 消息 Gift Message Example:**

```json
{
  "rid": "789012",
  "platform": "bilibili",
  "type": "Gift",
  "data": {
    "name": "用户B",
    "avatar": "https://example.com/avatar2.jpg",
    "item": "火箭 Rocket 🚀",
    "num": 1,
    "price": 500.0
  }
}
```

**Like 消息 Like Message Example:**

```json
{
  "rid": "112233",
  "platform": "kuaishou",
  "type": "Like",
  "data": {
    "name": "用户D",
    "avatar": "https://example.com/avatar4.jpg",
    "count": 10
  }
}
```

**EnterRoom 消息 Enter Room Message Example:**

```json
{
  "rid": "556677",
  "platform": "huya",
  "type": "EnterRoom",
  "data": {
    "name": "用户E",
    "avatar": "https://example.com/avatar5.jpg"
  }
}
```

**EndLive 消息 End Live Message Example:**

```json
{
  "rid": "666888",
  "platform": "douyin",
  "type": "EndLive",
  "data": {}
}
```

---

💡 **注意 Tips:** 此文档整合了 API 和 WebSocket 消息结构的所有信息，提供开发者一个清晰、一致的参考框架！