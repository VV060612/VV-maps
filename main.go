package main

import (
	"fmt"
	"log"
	"traffic-system/algo"
	"traffic-system/db"
	"traffic-system/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("=== 欢迎使用 VV Maps - 智能交通导航系统 ===")

	// 1. 初始化数据库
	// 连接 PostgreSQL，自动迁移表结构
	// 如果是第一次运行，会自动将 map_data.json 的数据导入数据库
	db.InitDB()

	// 2. 加载地图数据 (从数据库加载)
	// 注意：这里已经改为 LoadFromDB，不再读取本地 JSON 文件
	fmt.Println("正在从数据库构建图...")
	graph, err := algo.LoadFromDB()
	if err != nil {
		log.Fatalf("从数据库加载地图失败: %v", err)
	}
	fmt.Printf("地图加载成功! 节点数: %d\n", len(graph.Nodes))

	// 3. 将图对象传递给 handler (用于路径规划接口)
	handler.Graph = graph

	// 4. 初始化 Gin 引擎
	r := gin.Default()

	// 5. 配置路由
	setupRoutes(r)

	// 6. 启动服务器
	fmt.Println("\n服务器启动中...")
	fmt.Println("访问地址: http://localhost:8080")
	fmt.Println("前端页面: http://localhost:8080/static/")
	fmt.Println("API 文档:")
	fmt.Println("  - POST   /api/login          - 用户登录")
	fmt.Println("  - POST   /api/register       - 用户注册")
	fmt.Println("  - POST   /api/path/find      - 路径规划")
	fmt.Println("  - GET    /api/nodes          - 获取所有节点")
	fmt.Println("  - GET    /api/nodes/:id      - 获取指定节点")
	fmt.Println("  - GET    /api/nodes/search   - 搜索节点")
	fmt.Println("\n按 Ctrl+C 退出")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// setupRoutes 配置路由
func setupRoutes(r *gin.Engine) {
	// CORS 跨域中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 静态文件服务 - 提供前端页面
	r.Static("/static", "./static")

	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "ok",
		})
	})

	// 根路径重定向到前端页面
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/static/index.html")
	})

	// API 路由组
	api := r.Group("/api")
	{
		// 公开接口 (无需认证)
		api.POST("/login", handler.Login)
		api.POST("/register", handler.Register)

		// 地图相关接口
		api.POST("/path/find", handler.FindPath)
		api.GET("/nodes", handler.GetNodes)
		api.GET("/nodes/search", handler.SearchNodes)
		api.GET("/nodes/:id", handler.GetNodeByID)

		// 如果将来需要认证，可以解开下面的注释
		// authorized := api.Group("/")
		// authorizclaudeed.Use(handler.AuthMiddleware())
		// {
		//     authorized.POST("/path/find", handler.FindPath)
		// }
	}
}
