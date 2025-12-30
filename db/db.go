package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"traffic-system/model"

	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	// 从环境变量读取配置 (为了 Docker 部署方便)
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "vvuser")
	password := getEnvOrDefault("DB_PASSWORD", "vvpassword")
	dbname := getEnvOrDefault("DB_NAME", "vvtraffic")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		host, user, password, dbname, port,
	)

	// 带重试的数据库连接 (Docker 启动时数据库可能还没准备好)
	var err error
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("等待数据库就绪... (%d/%d): %v", i+1, maxRetries, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	// 自动迁移模式 (自动创建表结构)
	err = DB.AutoMigrate(&model.User{}, &model.Node{}, &model.Edge{})
	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 检查是否需要导入初始数据
	var nodeCount int64
	DB.Model(&model.Node{}).Count(&nodeCount)
	if nodeCount == 0 {
		log.Println("检测到数据库为空，正在导入 map_data.json...")
		if err := importMapData("map_data.json"); err != nil {
			log.Printf("警告: 导入地图数据失败: %v", err)
		} else {
			log.Println("地图数据导入成功!")
		}
	}

	log.Println("数据库连接并初始化成功！")
}

// getEnvOrDefault 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// importMapData 从 JSON 文件导入地图数据到数据库
func importMapData(filepath string) error {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 使用临时结构体解析 JSON (因为 JSON 中的 Modes 是 []string)
	var data struct {
		Meta  map[string]interface{} `json:"meta"`
		Nodes []model.Node           `json:"nodes"`
		Edges []struct {
			From   string   `json:"from"`
			To     string   `json:"to"`
			Dist   float64  `json:"dist"`
			Modes  []string `json:"modes"`
			LineID string   `json:"line_id,omitempty"`
			Desc   string   `json:"desc,omitempty"`
		} `json:"edges"`
	}

	if err := json.Unmarshal(file, &data); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	// 批量插入节点
	if len(data.Nodes) > 0 {
		if err := DB.CreateInBatches(data.Nodes, 100).Error; err != nil {
			return fmt.Errorf("插入节点失败: %w", err)
		}
		log.Printf("导入了 %d 个节点", len(data.Nodes))
	}

	// 批量插入边 (转换 Modes 为 pq.StringArray)
	if len(data.Edges) > 0 {
		edges := make([]model.Edge, len(data.Edges))
		for i, e := range data.Edges {
			edges[i] = model.Edge{
				From:   e.From,
				To:     e.To,
				Dist:   e.Dist,
				Modes:  pq.StringArray(e.Modes),
				LineID: e.LineID,
				Desc:   e.Desc,
			}
		}
		if err := DB.CreateInBatches(edges, 100).Error; err != nil {
			return fmt.Errorf("插入边失败: %w", err)
		}
		log.Printf("导入了 %d 条边", len(edges))
	}

	return nil
}
