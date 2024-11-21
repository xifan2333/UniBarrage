
# WebSocket 消息结构文档

此文档描述了通过 WebSocket 接收的统一消息结构。无论消息来源的平台是抖音、哔哩哔哩、快手、虎牙还是斗鱼，发送到 WebSocket 的数据都会遵循统一格式。

## 消息结构

每条消息都以 JSON 格式传递，包含以下字段：

```json
{
  "rid": "房间号",
  "platform": "来源平台",
  "type": "消息类型",
  "data": "消息内容数据"
}
```

### 字段说明

- `rid` (string)：房间号，用于标识直播间。
- `platform` (string)：来源平台，可能的值包括：
  - `douyin`
  - `bilibili`
  - `kuaishou`
  - `huya`
  - `douyu`
- `type` (string)：消息的类型，可能的值包括：
  - `Chat`：聊天消息
  - `Gift`：礼物消息
  - `Subscribe`：订阅/关注消息
  - `SuperChat`：超级聊天消息
  - `Like`：点赞消息
  - `EnterRoom`：进入房间消息
- `data` (object)：根据消息类型，包含不同的内容。

## 消息内容数据结构

### Chat（聊天消息）

```json
{
  "name": "发送者名称",
  "avatar": "发送者头像URL",
  "content": "聊天内容",
  "emoticon": ["表情URL列表"],
  "raw": "原始数据对象"
}
```

### Gift（礼物消息）

```json
{
  "name": "赠送者名称",
  "avatar": "赠送者头像URL",
  "item": "礼物名称",
  "num": "礼物数量",
  "price": "礼物单价",
  "giftIcon": "礼物图标URL",
  "raw": "原始数据对象"
}
```

### Subscribe（订阅/关注消息）

```json
{
  "name": "订阅者名称",
  "avatar": "订阅者头像URL",
  "item": "订阅项/关注类型",
  "num": "订阅次数",
  "price": "订阅单价",
  "raw": "原始数据对象"
}
```

### SuperChat（超级聊天消息）

```json
{
  "name": "发送者名称",
  "avatar": "发送者头像URL",
  "content": "超级聊天内容",
  "price": "超级聊天金额",
  "raw": "原始数据对象"
}
```

### Like（点赞消息） ⚠ 试验

```json
{
  "name": "点赞者名称",
  "avatar": "点赞者头像URL",
  "count": "点赞次数",
  "raw": "原始数据对象"
}
```

### EnterRoom（进入房间消息） ⚠ 试验

```json
{
  "name": "进入者名称",
  "avatar": "进入者头像URL",
  "raw": "原始数据对象"
}
```

### EndLive（结束直播消息） ⚠ 试验

```json
{
  "raw": "原始数据对象"
}
```

## 示例消息

### 示例 Chat 消息

```json
{
  "rid": "123456",
  "platform": "douyin",
  "type": "Chat",
  "data": {
    "name": "用户A",
    "avatar": "https://example.com/avatar.jpg",
    "content": "这是聊天消息",
    "emoticon": ["https://example.com/emoticon1.png"],
    "raw": {}
  }
}
```

### 示例 Gift 消息

```json
{
  "rid": "789012",
  "platform": "bilibili",
  "type": "Gift",
  "data": {
    "name": "用户B",
    "avatar": "https://example.com/avatar2.jpg",
    "item": "火箭",
    "num": 1,
    "price": 500.0,
    "giftIcon": "https://example.com/gift.png",
    "raw": {}
  }
}
```

### 示例 SuperChat 消息

```json
{
  "rid": "345678",
  "platform": "douyu",
  "type": "SuperChat",
  "data": {
    "name": "用户C",
    "avatar": "https://example.com/avatar3.jpg",
    "content": "支持主播！",
    "price": 100.0,
    "raw": {}
  }
}
```

### 示例 Like 消息

```json
{
  "rid": "112233",
  "platform": "kuaishou",
  "type": "Like",
  "data": {
    "name": "用户D",
    "avatar": "https://example.com/avatar4.jpg",
    "count": 10,
    "raw": {}
  }
}
```

### 示例 EnterRoom 消息

```json
{
  "rid": "556677",
  "platform": "huya",
  "type": "EnterRoom",
  "data": {
    "name": "用户E",
    "avatar": "https://example.com/avatar5.jpg",
    "raw": {}
  }
}
```

### 示例 EndLive 消息

```json
{
  "rid": "666888",
  "platform": "douyin",
  "type": "EndLive",
  "data": {
    "raw": {}
  }
}
```


本文档提供了统一消息结构的描述，以便开发者在处理来自不同平台的弹幕和事件时有一致的格式。
