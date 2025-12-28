package handler

import (
	"net/http"
	"traffic-system/algo"
	"traffic-system/model"

	"github.com/gin-gonic/gin"
)

// Graph 全局图对象 (应在 main 中初始化)
var Graph *algo.Graph

// PathRequest 路径规划请求
type PathRequest struct {
	StartID   string   `json:"start_id"`             // 起点节点 ID
	EndID     string   `json:"end_id"`               // 终点节点 ID
	StartLat  float64  `json:"start_lat,omitempty"`  // 起点纬度 (可选)
	StartLng  float64  `json:"start_lng,omitempty"`  // 起点经度 (可选)
	EndLat    float64  `json:"end_lat,omitempty"`    // 终点纬度 (可选)
	EndLng    float64  `json:"end_lng,omitempty"`    // 终点经度 (可选)
	Modes     []string `json:"modes" binding:"required"` // 交通方式: ["walk", "bike", "car", "bus", "subway"]
}

// PathResponse 路径规划响应
type PathResponse struct {
	Found         bool          `json:"found"`
	Path          []PathNode    `json:"path,omitempty"`
	Segments      []PathSegment `json:"segments,omitempty"`      // 路径段详情
	Distance      float64       `json:"distance,omitempty"`      // 总距离 (米)
	EstimatedTime float64       `json:"estimated_time,omitempty"` // 预计时间 (秒)
	Message       string        `json:"message,omitempty"`
}

// PathNode 路径节点信息
type PathNode struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Type string  `json:"type"`
}

// PathSegment 路径段信息
type PathSegment struct {
	FromID   string   `json:"from_id"`
	FromName string   `json:"from_name"`
	ToID     string   `json:"to_id"`
	ToName   string   `json:"to_name"`
	Distance float64  `json:"distance"`
	Time     float64  `json:"time"`      // 预计时间 (秒)
	Modes    []string `json:"modes"`     // 可用的交通方式
	UsedMode string   `json:"used_mode"` // 实际使用的交通方式
	LineID   string   `json:"line_id,omitempty"`
	Desc     string   `json:"desc,omitempty"`
}

// FindPath 路径规划接口
func FindPath(c *gin.Context) {
	var req PathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	if Graph == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "地图数据未加载"})
		return
	}

	// 如果提供了坐标，找到最近的节点
	startID := req.StartID
	endID := req.EndID

	if req.StartLat != 0 && req.StartLng != 0 {
		nearestStart := Graph.FindNearestNode(req.StartLat, req.StartLng)
		if nearestStart != nil {
			startID = nearestStart.ID
		}
	}

	if req.EndLat != 0 && req.EndLng != 0 {
		nearestEnd := Graph.FindNearestNode(req.EndLat, req.EndLng)
		if nearestEnd != nil {
			endID = nearestEnd.ID
		}
	}

	// 验证起点和终点
	if startID == "" || endID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "起点或终点未指定"})
		return
	}

	if Graph.Nodes[startID] == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "起点不存在: " + startID})
		return
	}

	if Graph.Nodes[endID] == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "终点不存在: " + endID})
		return
	}

	// 解析交通方式
	modeMask := model.ParseModes(req.Modes)
	if modeMask == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未指定有效的交通方式"})
		return
	}

	// 执行路径规划
	result := Graph.Dijkstra(startID, endID, modeMask)

	if !result.Found {
		c.JSON(http.StatusOK, PathResponse{
			Found:   false,
			Message: "未找到符合条件的路径",
		})
		return
	}

	// 构建路径节点信息
	pathNodes := make([]PathNode, 0, len(result.Path))
	for _, nodeID := range result.Path {
		node := Graph.Nodes[nodeID]
		if node != nil {
			pathNodes = append(pathNodes, PathNode{
				ID:   node.ID,
				Name: node.Name,
				Lat:  node.Lat,
				Lng:  node.Lng,
				Type: node.Type,
			})
		}
	}

	// 构建路径段信息（包含节点名称）
	segments := make([]PathSegment, 0, len(result.Segments))
	for _, seg := range result.Segments {
		fromNode := Graph.Nodes[seg.FromID]
		toNode := Graph.Nodes[seg.ToID]
		fromName, toName := seg.FromID, seg.ToID
		if fromNode != nil {
			fromName = fromNode.Name
		}
		if toNode != nil {
			toName = toNode.Name
		}
		segments = append(segments, PathSegment{
			FromID:   seg.FromID,
			FromName: fromName,
			ToID:     seg.ToID,
			ToName:   toName,
			Distance: seg.Distance,
			Time:     seg.Time,
			Modes:    seg.Modes,
			UsedMode: seg.UsedMode,
			LineID:   seg.LineID,
			Desc:     seg.Desc,
		})
	}

	c.JSON(http.StatusOK, PathResponse{
		Found:         true,
		Path:          pathNodes,
		Segments:      segments,
		Distance:      result.Distance,
		EstimatedTime: result.EstimatedTime,
		Message:       "路径规划成功",
	})
}

// GetNodes 获取所有节点信息
func GetNodes(c *gin.Context) {
	if Graph == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "地图数据未加载"})
		return
	}

	nodes := make([]PathNode, 0, len(Graph.NodeList))
	for _, node := range Graph.NodeList {
		nodes = append(nodes, PathNode{
			ID:   node.ID,
			Name: node.Name,
			Lat:  node.Lat,
			Lng:  node.Lng,
			Type: node.Type,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(nodes),
		"nodes": nodes,
	})
}

// GetNodeByID 根据 ID 获取节点信息
func GetNodeByID(c *gin.Context) {
	nodeID := c.Param("id")

	if Graph == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "地图数据未加载"})
		return
	}

	node := Graph.Nodes[nodeID]
	if node == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}

	c.JSON(http.StatusOK, PathNode{
		ID:   node.ID,
		Name: node.Name,
		Lat:  node.Lat,
		Lng:  node.Lng,
		Type: node.Type,
	})
}

// SearchNodes 搜索节点 (根据名称模糊匹配)
func SearchNodes(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少搜索关键词"})
		return
	}

	if Graph == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "地图数据未加载"})
		return
	}

	results := make([]PathNode, 0)
	for _, node := range Graph.NodeList {
		// 简单的名称匹配 (可以改进为更复杂的搜索算法)
		if contains(node.Name, query) || contains(node.ID, query) {
			results = append(results, PathNode{
				ID:   node.ID,
				Name: node.Name,
				Lat:  node.Lat,
				Lng:  node.Lng,
				Type: node.Type,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":  query,
		"count":  len(results),
		"results": results,
	})
}

// contains 检查字符串是否包含子串 (不区分大小写)
func contains(s, substr string) bool {
	// 简单的包含检查 (可以使用 strings.Contains)
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
