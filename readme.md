# UniBarrage 文档 📜✨

**UniBarrage** 是一款开源的高性能实时代理工具，专为开发者设计，旨在统一多平台（如抖音、哔哩哔哩、快手、斗鱼、虎牙）直播弹幕数据的采集、解析和转发。通过标准化的 WebSocket 消息协议和灵活的 API 接口，UniBarrage 将分散的多平台弹幕数据统一为一致的格式，帮助开发者更高效地构建跨平台互动功能。  

**UniBarrage** is an **open-source** high-performance real-time proxy tool designed for developers, focusing on unifying the collection, parsing, and forwarding of live barrage data from multiple platforms (e.g., Douyin, Bilibili, Kuaishou, Douyu, and Huya). With standardized WebSocket message protocols and flexible API interfaces, UniBarrage consolidates disparate multi-platform barrage data into a consistent format, enabling developers to build cross-platform interactive features more efficiently.

---

### **核心特点 🌟 | Core Features**

- **多平台支持 🖥️**：同时兼容多种主流直播平台，无需针对不同平台重复开发。
-
  **Multi-Platform Support 🖥️**: Compatible with various mainstream live streaming platforms, eliminating the need for repetitive development.
  
- **实时性强 ⚡**：通过高性能 WebSocket 提供毫秒级延迟的弹幕转发服务。
-
  **High Real-Time Performance ⚡**: Leverages high-performance WebSocket to deliver millisecond-latency barrage forwarding services.
  
- **统一数据结构 🔄**：简化开发者工作，轻松实现跨平台互动功能。
-
  **Unified Data Structure 🔄**: Simplifies development by enabling seamless cross-platform interactive features.
  
- **灵活扩展性 🔧**：支持多种启动参数和 API 调用，满足不同场景的定制化需求。
-
  **Flexible Extensibility 🔧**: Supports diverse startup parameters and API calls to meet the customization needs of various scenarios.
  
- **小巧的二进制体积 📦**：高度优化的代码，编译后的二进制文件小于 10MB，便于分发和部署。
-
  **Compact Binary Size 📦**: Highly optimized code with a compiled binary size of less than 10MB, making it easy to distribute and deploy.
  
- **100% 不丢失弹幕消息 🛡️**：得益于 Go 的通道机制和高性能队列，确保每条弹幕都能被安全、高效地处理和转发。
-
  **100% No Message Loss 🛡️**: Powered by Go's channel mechanism and high-performance queues, ensuring every message is handled and forwarded safely and efficiently.
  
- **开源与社区支持 🌍**：完全开源，拥有活跃的社区支持，开发者可轻松贡献或扩展功能。
-
  **Open Source with Community Support 🌍**: Fully open-source with an active community, allowing developers to easily contribute or extend its capabilities.

---

**UniBarrage** 不仅适用于个人开发者快速构建项目，也为团队协作提供了一个稳定、可靠的基础设施，助力直播生态的创新发展。无论是实现跨平台弹幕墙、智能弹幕分析，还是与观众实时互动，UniBarrage 都是您的得力助手。  
  
**UniBarrage** is not only ideal for individual developers to rapidly build projects but also provides a stable and reliable infrastructure for team collaboration, driving innovation in the live streaming ecosystem. Whether it’s for building cross-platform barrage walls, performing intelligent barrage analysis, or engaging in real-time interaction with the audience, UniBarrage is your trusted assistant.

---

## 一、文档大纲

1. [UniBarrage 简介 🌟](#unibarrage-intro)
2. [API 接口文档 🌐](#api-documentation)
    - [启动参数 ⚙️](#startup-parameters)
    - [API 列表 📬](#api-list)
3. [WebSocket 消息结构 📡](#websocket-message-structure)
    - [消息字段说明 📜](#message-field-descriptions)
    - [消息类型及示例 🧩](#message-types-and-examples)
4. [错误码参考表 🚨](#error-codes)

---

<a id="unibarrage-intro"></a>

## UniBarrage 简介 🌟

UniBarrage 是一个帮助开发者统一处理多平台直播弹幕数据的工具，支持高性能实时代理及标准化转发。

- **支持平台**: 抖音、哔哩哔哩、快手、斗鱼、虎牙
- **核心功能**: 统一格式的 WebSocket 消息流和灵活的 API 接口

---

<a id="api-documentation"></a>

## API 接口文档 🌐

<a id="startup-parameters"></a>

### 启动参数 ⚙️

| 参数名          | 类型       | 默认值         | 描述                      |
|--------------|----------|-------------|-------------------------|
| `-wsHost`    | `string` | `127.0.0.1` | WebSocket 服务的主机地址       |
| `-wsPort`    | `int`    | `7777`      | WebSocket 服务的端口号        |
| `-apiHost`   | `string` | `127.0.0.1` | API 服务的主机地址             |
| `-apiPort`   | `int`    | `8080`      | API 服务的端口号              |
| `-useProxy`  | `bool`   | `false`     | 是否启用代理服务                |
| `-authToken` | `string` | `""`        | Bearer Token (仅 API 使用) |

#### 示例命令 🛠️

```bash
# 示例：启动 UniBarrage 服务
./UniBarrage -wsHost 127.0.0.1 -wsPort 7777 -apiHost 127.0.0.1 -apiPort 8080 -useProxy
```

---

<a id="api-list"></a>

### API 列表 📬

#### 欢迎接口 Welcome Endpoint 👋

- **URL**: `/`
- **方法 Method**: `GET`
- **描述 Description**: 返回欢迎信息，验证服务是否启动。

**响应示例 Response Example:**

```json
{
  "code": 200,
  "message": "Hello, UniBarrage!",
  "data": null
}
```

#### 获取所有服务状态 Get All Services Status 🔄

- **URL**: `/all`
- **方法 Method**: `GET`
- **描述 Description**: 获取所有正在运行的服务状态。

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

#### 获取指定平台的所有服务 Get Services for Specific Platform 🔍

- **URL**: `/{platform}`
- **方法 Method**: `GET`
- **描述 Description**: 获取指定平台的所有服务。

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

#### 获取单个服务状态 Get Single Service Status 🧐

- **URL**: `/{platform}/{roomId}`
- **方法 Method**: `GET`
- **描述 Description**: 获取指定房间的服务状态。

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

#### 启动服务 Start Service 🚀

- **URL**: `/{platform}`
- **方法 Method**: `POST`
- **描述 Description**: 启动指定平台的直播服务。

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

#### 停止服务 Stop Service 🛑

- **URL**: `/{platform}/{roomId}`
- **方法 Method**: `DELETE`
- **描述 Description**: 停止指定平台房间的服务。

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

<a id="websocket-message-structure"></a>

## WebSocket 消息结构 📡

<a id="message-field-descriptions"></a>

### 消息字段说明 📜

```text
- rid: 房间号 Room ID
- platform: 来源平台 Platform (如 Douyin, Bilibili)
- type: 消息类型 Message Type
  - Chat: 聊天消息
  - Gift: 礼物消息
  - Like: 点赞消息
  - EnterRoom: 进入房间消息
  - Subscribe: 订阅消息
  - SuperChat: 超级聊天消息
  - EndLive: 结束直播消息
```

<a id="message-types-and-examples"></a>

### 消息类型及示例 🧩

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

<a id="error-codes"></a>

## 错误码参考表 🚨

| 错误码 Code | 描述 Description            |
|----------|---------------------------|
| `200`    | ✅ 请求成功 Request successful |
| `201`    | ✅ 服务创建成功 Service created  |
| `400`    | ⚠️ 请求参数错误 Bad request     |
| `404`    | ❌ 服务未找到 Service not found |
| `500`    | ❌ 服务器内部错误 Internal error  |

---

💡 **提示 Tips:** 此文档整合了 API 和 WebSocket 消息结构的所有信息，提供开发者一个清晰、一致的参考框架！

