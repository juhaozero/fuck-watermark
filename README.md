# fuck-watermark

基于 Gin 框架短视频去水印解析 API 服务

## 支持平台

| 平台 | 路由 | 状态 |
|------|------|------|
| 抖音 | `/api/douyin` | ✅ |
| 快手 | `/api/kuaishou` | ✅ |
| 小红书 | `/api/xiaohongshu` 或 `/api/xhsjx` | ✅ |
| 哔哩哔哩 | `/api/bilibili` | ✅ |
| 微博 | `/api/weibo` | ✅ |
| 豆包 AI 视频 | `/api/doubao` | ✅ |
| 自动识别 | `/api/parse` | ✅ |

## 快速开始

### 本地运行

```bash
cp config.toml.example config.toml
# 按需编辑 config.toml（端口、鉴权等）
go mod tidy
go run ./cmd/server
```

服务默认监听 `http://0.0.0.0:8080`，配置文件默认为当前目录下的 `config.toml`。

### Docker 构建与运行

```bash
cp config.toml.example config.toml
# 按需编辑 config.toml

# 构建镜像
docker build -t fuck-watermark:latest .

# 直接运行（挂载配置文件）
docker run -d \
  --name fuck-watermark \
  -p 8080:8080 \
  -v "$(pwd)/config.toml:/app/config.toml:ro" \
  fuck-watermark:latest
```

### Docker Compose 部署（推荐）

```bash
# 镜像
ghcr.io/juhaozero/fuck-watermark:latest

cp config.toml.example config.toml
# 按需编辑 config.toml
```

## API 使用

### 健康检查

```http
GET /health
```

### 自动识别平台解析

```http
GET /api/parse?url=https://v.douyin.com/xxxx/
```

### 指定平台解析

```http
GET /api/douyin?url=https://v.douyin.com/xxxx/
GET /api/kuaishou?url=https://v.kuaishou.com/xxxx
GET /api/xiaohongshu?url=https://xhslink.com/xxxx
GET /api/bilibili?url=https://www.bilibili.com/video/BVxxxx
GET /api/weibo?url=https://weibo.com/tv/show/xxxx
GET /api/doubao?url=https://www.doubao.com/video-sharing?share_id=xxx&video_id=xxx
```

支持 `GET`、`POST`（表单或 JSON）。除 `url` 外，可传 `cookie` 作为各平台登录态（可选，提高解析成功率）。

```http
GET /api/douyin?url=https://v.douyin.com/xxxx/&cookie=your_platform_cookie
POST /api/parse
Content-Type: application/json

{"url":"https://v.douyin.com/xxxx/","cookie":"your_platform_cookie"}
```

### 响应示例

```json
{
  "code": 200,
  "msg": "解析成功",
  "data": {
    "platform": "douyin",
    "type": "video",
    "title": "视频标题",
    "desc": "视频描述",
    "author": {
      "name": "作者名称",
      "id": "123456789",
      "avatar": "https://example.com/avatar.jpg"
    },
    "cover": "https://example.com/cover.jpg",
    "url": "https://example.com/video.mp4",
    "video_backup": [
      { "url": "https://example.com/video_hd.mp4", "label": "720p", "quality": "hd" }
    ],
    "video_id": "7123456789",
    "images": [],
    "live_photo": [],
    "music": {
      "title": "背景音乐",
      "author": "音乐作者",
      "url": "https://example.com/music.mp3",
      "cover": "https://example.com/music_cover.jpg"
    },
    "parts": [],
    "stats": {
      "like_count": 1000,
      "play_count": 5000
    }
  }
}
```

所有平台成功响应均使用统一的 `VideoData` 结构。B 站多 P 视频见 `parts` 字段；微博多清晰度见 `video_backup`（含 `label`/`quality`）。

错误响应不含 `data` 字段：

```json
{
  "code": 400,
  "msg": "请输入有效的链接"
}
```

### 状态码

| code | 说明 |
|------|------|
| 200 | 解析成功 |
| 400 | 参数错误或不支持的平台 |
| 404 | 解析失败 |
| 500 | 请求/服务器错误 |

## 配置文件

复制 `config.toml.example` 为 `config.toml` 后修改。也可通过 `-config` 指定路径：

```bash
go run ./cmd/server -config /path/to/config.toml
```

示例（`config.toml`）：

```toml
[server]
addr = 8080
request_timeout = 15

[weibo]
proxy_base = ""

[security]
api_key = ""
allow_origins = ["*"]
max_body_bytes = 4096

[rate_limit]
enabled = true
requests_per_minute = 60
burst = 10
```

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `server.addr` | `8080` | 服务监听端口 |
| `server.request_timeout` | `15` | 上游请求超时（秒） |
| `weibo.proxy_base` | 空 | 微博视频代理基址 |
| `security.api_key` | 空 | API 密钥；配置后 `/api/*` 需携带 `X-API-Key` 或 `Authorization: Bearer` |
| `security.allow_origins` | `["*"]` | CORS 允许来源 |
| `security.max_body_bytes` | `16384` | 请求体大小上限（含 cookie 字段） |
| `rate_limit.enabled` | `true` | 是否启用 IP 限流 |
| `rate_limit.requests_per_minute` | `60` | 每 IP 每分钟请求数 |
| `rate_limit.burst` | `10` | 突发请求上限 |



### Nginx 反向代理示例

```nginx
server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 30s;
    }
}
```

## 项目结构

```
├── cmd/server/main.go          # 入口（加载配置、启动服务）
├── config.toml.example         # 配置模板
├── internal/
│   ├── config/                 # TOML 配置加载
│   ├── endpoints/              # 各平台上游 URL 常量
│   ├── handler/                # HTTP 处理器
│   ├── httputil/               # 出站 HTTP 客户端
│   ├── middleware/             # CORS / 鉴权 / 限流 / 安全头
│   ├── model/                  # 统一响应模型
│   ├── parser/                 # 各平台解析器（实现 Parser 接口）
│   ├── platform/               # 平台注册表与路由别名
│   ├── server/                 # 服务组装与路由挂载
│   └── urlutil/                # URL 校验（SSRF 防护）
├── Dockerfile
├── docker-compose.yml
└── README.md
```

### 扩展新平台

1. 在 `internal/parser/{name}/` 实现 `parser.Parser` 接口
2. 在 `internal/endpoints/endpoints.go` 添加上游 URL 常量
3. 在 `internal/platform/bootstrap.go` 的 `DefaultDescriptors()` 追加一条 `Descriptor`

路由会自动注册，无需修改 `main.go`。

