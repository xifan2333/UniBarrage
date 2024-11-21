# UniBarrage API 接口文档

## 概述

`UniBarrage` 是一个高性能实时代理和统一弹幕数据转发器，用于支持多平台（如抖音、哔哩哔哩、快手、斗鱼、虎牙）的直播弹幕转发。以下是
API 的详细接口文档。

## 二进制启动参数

在运行 `UniBarrage` 二进制文件时，可以使用以下命令行参数来配置服务。

### 命令行参数列表

| 参数名               | 类型     | 默认值         | 描述                         |
|-------------------|--------|-------------|----------------------------|
| `-wsHost`         | string | `127.0.0.1` | WebSocket 服务的主机地址          |
| `-wsPort`         | int    | `7777`      | WebSocket 服务的端口号           |
| `-apiHost`        | string | `127.0.0.1` | API 服务的主机地址                |
| `-apiPort`        | int    | `8080`      | API 服务的端口号                 |
| `-useProxy`       | bool   | `false`     | 是否启用代理服务                   |
| `-proxyHost`      | string | `127.0.0.1` | 代理服务的主机地址                  |
| `-proxyPort`      | int    | `8888`      | 代理服务的端口号                   |
| `-certFile`       | string | `""`        | SSL/TLS 证书文件路径             |
| `-keyFile`        | string | `""`        | SSL/TLS 私钥文件路径             |
| `-allowedOrigins` | string | `"*"`       | 允许跨域请求的来源列表（用逗号分隔）         |
| `-authToken`      | string | `""`        | 用于验证的 Bearer Token (仅API)  |
| `-logLevel`       | int    | `0`         | 日志等级 (0: 默认, 1: 简洁, 2: 静默) |

### 示例命令

```bash
./UniBarrage -wsHost 192.168.1.10 -wsPort 9000 -apiHost 192.168.1.10 -apiPort 8081 -useProxy -proxyHost 192.168.1.10 -proxyPort 9090
```

上述命令会启动 `UniBarrage` 服务，设置 WebSocket 服务和 API 服务在指定的主机和端口上运行，并启用代理服务。

## 基础 URL

`http://{host}:{port}/api/v1`

## 接口列表

### 1. 欢迎接口

- **URL**: `/`
- **方法**: `GET`
- **描述**: 返回欢迎信息，验证服务是否启动。

#### 响应示例

```json
{
  "code": 200,
  "message": "Hello, UniBarrage!",
  "data": null
}
```

### 2. 获取所有服务状态

- **URL**: `/all`
- **方法**: `GET`
- **描述**: 获取所有正在运行的服务状态。

#### 响应示例

```json
{
  "code": 200,
  "message": "获取成功",
  "data": [
    {
      "platform": "douyin",
      "rid": "123456",
      "is_running": true
    },
    {
      "platform": "bilibili",
      "rid": "789012",
      "is_running": true
    }
  ]
}
```

### 3. 获取指定平台的所有服务

- **URL**: `/{platform}`
- **方法**: `GET`
- **描述**: 获取指定平台的所有服务。

#### 请求参数

- `platform`（路径参数）：需要查询的直播平台名称，例如 `douyin`, `bilibili`, `kuaishou`, `douyu`, `huya`。

#### 响应示例

```json
{
  "code": 200,
  "message": "获取成功",
  "data": [
    {
      "platform": "douyin",
      "rid": "123456",
      "is_running": true
    }
  ]
}
```

### 4. 获取单个服务状态

- **URL**: `/{platform}/{roomId}`
- **方法**: `GET`
- **描述**: 获取指定房间的服务状态。

#### 请求参数

- `platform`（路径参数）：直播平台名称。
- `roomId`（路径参数）：房间 ID。

#### 响应示例

```json
{
  "code": 200,
  "message": "获取成功",
  "data": {
    "platform": "douyin",
    "rid": "123456",
    "is_running": true
  }
}
```

### 5. 启动服务

- **URL**: `/{platform}`
- **方法**: `POST`
- **描述**: 启动指定平台的直播服务。

#### 请求体示例

```json
{
  "rid": "123456",
  "cookie": "可选的登录cookie"
}
```

#### 请求参数

- `platform`（路径参数）：直播平台名称。
- `rid`（请求体参数）：房间 ID。
- `cookie`（请求体参数，可选）：用于需要登录的服务。

#### 响应示例

```json
{
  "code": 201,
  "message": "服务启动成功",
  "data": {
    "platform": "douyin",
    "rid": "123456"
  }
}
```

### 6. 停止服务

- **URL**: `/{platform}/{roomId}`
- **方法**: `DELETE`
- **描述**: 停止指定平台房间的服务。

#### 请求参数

- `platform`（路径参数）：直播平台名称。
- `roomId`（路径参数）：房间 ID。

#### 响应示例

```json
{
  "code": 200,
  "message": "服务已停止",
  "data": {
    "platform": "douyin",
    "rid": "123456"
  }
}
```

### 错误响应示例

```json
{
  "code": 404,
  "message": "服务未找到",
  "data": null
}
```

## 错误码

- `200` - 请求成功
- `201` - 服务创建成功
- `400` - 请求参数错误
- `404` - 服务未找到
- `500` - 服务器内部错误

## 示例服务状态

`ServiceStatus` 结构体：

```json
{
  "platform": "douyin",
  "rid": "123456",
  "is_running": true
}
```
