# VV Maps - 智能交通导航系统

VV Maps 是一款基于 Go 语言开发的多模态路径规划后端系统。本项目以**郑州高新区（重点涵盖河南工业大学与郑州大学区域）**的真实路网为蓝本，实现了一个高效、精准的路径搜索引擎。

## 技术栈

| 类别 | 技术 |
|------|------|
| Backend | Go 1.24+ |
| Framework | Gin Web Framework |
| Database | PostgreSQL 15 |
| ORM | GORM |
| Algorithm | Dijkstra + Min-priority Queue |
| Frontend | Vue3 + Leaflet.js + OpenStreetMap |
| Container | Docker + Docker Compose |

## 快速开始

### 方式一：Docker 部署（推荐）

```bash
# 克隆项目
git clone <repository-url>
cd traffic-system

# 一键启动（包含数据库和应用）
docker compose up -d

# 查看日志
docker compose logs -f app

# 停止服务
docker compose down
```

启动后访问：
- 前端页面：http://localhost:8080/static/
- 健康检查：http://localhost:8080/ping

### 方式二：本地开发

**前置条件：**
- Go 1.24+
- PostgreSQL 15+

**步骤：**

```bash
# 1. 启动 PostgreSQL（可以用 Docker）
docker run -d --name vv-postgres \
  -e POSTGRES_USER=vvuser \
  -e POSTGRES_PASSWORD=vvpassword \
  -e POSTGRES_DB=vvtraffic \
  -p 5432:5432 \
  postgres:15-alpine

# 2. 设置环境变量
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=vvuser
export DB_PASSWORD=vvpassword
export DB_NAME=vvtraffic

# 3. 运行应用
go run .
```

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DB_HOST` | 数据库主机 | localhost |
| `DB_PORT` | 数据库端口 | 5432 |
| `DB_USER` | 数据库用户 | vvuser |
| `DB_PASSWORD` | 数据库密码 | vvpassword |
| `DB_NAME` | 数据库名 | vvtraffic |
| `GIN_MODE` | Gin 运行模式 | debug |

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/ping` | 健康检查 |
| POST | `/api/login` | 用户登录 |
| POST | `/api/register` | 用户注册 |
| POST | `/api/path/find` | 路径规划 |
| GET | `/api/nodes` | 获取所有节点 |
| GET | `/api/nodes/:id` | 获取指定节点 |
| GET | `/api/nodes/search` | 搜索节点 |

### 路径规划示例

```bash
curl -X POST http://localhost:8080/api/path/find \
  -H "Content-Type: application/json" \
  -d '{
    "start_id": "haut_gate_s",
    "end_id": "zzu_gate_n",
    "modes": ["walk", "bus"]
  }'
```

## 项目结构

```
.
├── algo/                 # 核心算法 (Graph加载、Dijkstra实现)
├── db/                   # 数据库初始化与连接
├── handler/              # Web 接口处理
├── model/                # 数据模型 (Node, Edge, User)
├── static/               # 前端页面 (Vue3 + Leaflet)
├── utils/                # 工具函数 (Haversine距离计算、密码加密)
├── map_data.json         # 路网数据（首次启动自动导入数据库）
├── Dockerfile            # Docker 镜像构建
├── docker-compose.yml    # Docker Compose 编排
├── go.mod                # Go 模块定义
└── main.go               # 程序入口
```

## 核心算法

### 路网建模

- **节点 (Node)**：地标、路口、公交站、地铁站
- **边 (Edge)**：连接两个节点的通道，包含距离和支持的交通模式
- **双向/单向**：普通道路自动生成反向边，公交/地铁遵循单向线路

### 多模态位掩码

采用二进制位掩码实现毫秒级权限判断：

```go
const (
    ModeWalk   = 1 << 0  // 步行
    ModeBike   = 1 << 1  // 骑行
    ModeCar    = 1 << 2  // 驾车
    ModeBus    = 1 << 3  // 公交
    ModeSubway = 1 << 4  // 地铁
)

// 快速判断通行权限
if edge.ModeMask & userModeMask != 0 {
    // 可通行
}
```

### 时间成本函数

不仅计算距离，更计算**时间**：

- **行驶时间** = 距离 / 平均速度
- **等待成本**：骑行/驾车考虑取车时间，公交/地铁考虑等待时间
- **换乘优化**：同线连续站点不重复计算等待成本

## 数据初始化

首次启动时，系统会自动：
1. 连接 PostgreSQL（带重试机制，适配 Docker 启动顺序）
2. 自动创建 `users`、`nodes`、`edges` 表
3. 检测到数据为空时，自动从 `map_data.json` 导入路网数据

## 开发指南

```bash
# 构建
go build -o main .

# 测试
go test ./...

# Docker 重新构建
docker compose build --no-cache
docker compose up -d
```

## License

MIT
