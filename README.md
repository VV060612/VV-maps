------

# VV Maps - 智能交通导航系统 (Zhengzhou Pathfinding)

VV Maps 是一款基于 Go 语言开发的多模态路径规划后端系统。本项目以**郑州高新区（重点涵盖河南工业大学与郑州大学区域）**的真实路网为蓝本，实现了一个高效、精准的路径搜索引擎。

## 🚀 项目核心亮点：Dijkstra 算法逻辑实现

本项目不仅实现了基础的 Dijkstra 最短路径算法，还针对真实的城市交通场景进行了深度定制。

### 1. 路网建模：节点 (Nodes) 与 边 (Edges)

我们将郑州区域的地理信息抽象为图结构：

- **节点 (Node)**：涵盖了地标（如河工大南门）、路口（如莲花街/长椿路）、公交站以及地铁站。
- **边 (Edge)**：连接两个节点的通道。每条边不仅包含物理距离（Distance），还定义了支持的**交通模式 (Modes)**。
- **双向/单向逻辑**：算法在加载数据时，会自动为普通道路（步行/骑行/驾车）生成反向边，而对公交和地铁则严格遵循单向线路逻辑。

### 2. 多模态路径规划：位掩码 (Bitmask) 优化

为了实现毫秒级的权限判断，项目采用了二进制位掩码技术：

- 通过 `1<<0` 到 `1<<4` 分别代表 **步行、骑行、驾车、公交、地铁**。
- 在 Dijkstra 扩展邻居节点时，只需进行一次位运算 `edge.ModeMask & userModeMask != 0` 即可快速过滤掉用户无法通行的路径。

### 3. 时间成本函数：不仅仅是距离

传统的算法只算距离，而 VV Maps 算的是**时间**。我们在 Dijkstra 中引入了动态权重计算：

- **行驶时间**：`距离 / 该模式下的平均速度`。
- **等待/准备成本**：
  - **骑行/驾车**：考虑了找车、解锁或取车的时间开销。
  - **公交/地铁**：引入了平均等待时间。更智能的是，算法能识别“同线连续站点”，此时不重复计算等待成本。

### 4. 算法流程

1. **初始化**：使用 `container/heap` 构建最小优先队列，以时间成本（Cost）为排序标准。
2. **松弛操作 (Relaxation)**：从起点出发，不断更新到达每个节点的最短时间成本。
3. **结果回溯**：找到终点后，通过前驱记录（Prev）回溯出完整的节点序列和分段详情（Segments）。

## 🛠️ 技术栈

- **Backend**: Go (Golang)
- **Framework**: Gin Web Framework
- **Algorithm**: Dijkstra + Min-priority Queue
- **Database**: JSON File Data Store
- **Map Interface**: Leaflet.js + OpenStreetMap

## 📂 项目目录结构

Plaintext

```
.
├── algo/             # 核心算法逻辑 (Graph加载、Dijkstra实现)
├── handler/          # Web 接口处理 (路径规划、节点查询、用户认证)
├── model/            # 数据模型定义 (Node, Edge, User)
├── static/           # 前端可视化页面 (基于 Vue3 + Leaflet)
├── utils/            # 工具函数 (Haversine经纬度计算、密码加密)
├── map_data.json     # 郑州高新区路网数据
└── main.go           # 程序入口，启动 Gin 服务
```

## 🚦 如何启动

1. 确保本地已安装 Go 环境。

2. 克隆项目并进入目录。

3. 运行命令：

   Bash

   ```
   go run main.go
   ```

4. 打开浏览器访问：`http://localhost:8080/static/index.html`。