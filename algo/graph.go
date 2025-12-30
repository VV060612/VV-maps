package algo

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"traffic-system/db" // 引入数据库包
	"traffic-system/model"
	"traffic-system/utils"
)

// Graph 图结构，用于路径规划
type Graph struct {
	Nodes    map[string]*model.Node   // 节点字典 (ID -> Node)
	AdjList  map[string][]*model.Edge // 邻接表 (ID -> 边列表)
	NodeList []model.Node             // 节点列表 (用于遍历)
}

// NewGraph 创建一个空的图
func NewGraph() *Graph {
	return &Graph{
		Nodes:   make(map[string]*model.Node),
		AdjList: make(map[string][]*model.Edge),
	}
}

// LoadFromDB 从数据库加载数据构建图 (新增函数)
func LoadFromDB() (*Graph, error) {
	g := NewGraph()

	// 1. 从数据库查询所有节点
	var dbNodes []model.Node
	// 使用 db.DB 直接查询
	if err := db.DB.Find(&dbNodes).Error; err != nil {
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}

	// 将节点填入图
	for i := range dbNodes {
		// 注意：这里要取地址，或者拷贝一份，避免循环变量复用问题
		node := dbNodes[i]
		g.Nodes[node.ID] = &node
		g.NodeList = append(g.NodeList, node)
	}

	// 2. 从数据库查询所有边
	var dbEdges []model.Edge
	if err := db.DB.Find(&dbEdges).Error; err != nil {
		return nil, fmt.Errorf("查询边失败: %w", err)
	}

	// 将边填入邻接表
	for i := range dbEdges {
		edge := &dbEdges[i]

		// 重新计算 ModeMask (因为数据库只存了字符串数组 ["walk", "car"])
		edge.ModeMask = model.ParseModes(edge.Modes)

		// 加入邻接表
		g.AdjList[edge.From] = append(g.AdjList[edge.From], edge)

		// 3. 处理双向道路 (自动生成反向边)
		// 逻辑：如果支持 walk/bike/car，则认为是双向的，自动加一条反向边到内存
		bidirectionalMask := model.ModeWalk | model.ModeBike | model.ModeCar
		if edge.ModeMask&bidirectionalMask != 0 {
			// 创建反向边 (仅在内存中存在，不写回数据库)
			reverseEdge := &model.Edge{
				From:     edge.To,
				To:       edge.From,
				Dist:     edge.Dist,
				Modes:    getBidirectionalModes(edge.Modes),
				ModeMask: edge.ModeMask & bidirectionalMask,
				Desc:     edge.Desc + " (反向)",
			}
			g.AdjList[edge.To] = append(g.AdjList[edge.To], reverseEdge)
		}
	}

	log.Printf("成功从数据库加载图: %d 个节点, %d 条基础边", len(g.Nodes), len(dbEdges))
	return g, nil
}

// LoadFromJSON 保留旧方法作为备份 (可选)
func LoadFromJSON(filepath string) (*Graph, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var data model.MapData
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	g := NewGraph()

	for i := range data.Nodes {
		node := &data.Nodes[i]
		g.Nodes[node.ID] = node
		g.NodeList = append(g.NodeList, *node)
	}

	for i := range data.Edges {
		edge := &data.Edges[i]
		edge.ModeMask = model.ParseModes(edge.Modes)

		if edge.Dist == 0 {
			from := g.Nodes[edge.From]
			to := g.Nodes[edge.To]
			if from != nil && to != nil {
				p1 := model.Point{Lat: from.Lat, Lng: from.Lng}
				p2 := model.Point{Lat: to.Lat, Lng: to.Lng}
				edge.Dist = utils.HaversineDistance(p1, p2)
			}
		}

		g.AdjList[edge.From] = append(g.AdjList[edge.From], edge)

		bidirectionalMask := model.ModeWalk | model.ModeBike | model.ModeCar
		if edge.ModeMask&bidirectionalMask != 0 {
			reverseExists := false
			for _, existingEdge := range g.AdjList[edge.To] {
				if existingEdge.From == edge.To && existingEdge.To == edge.From {
					reverseExists = true
					break
				}
			}
			if !reverseExists {
				reverseEdge := &model.Edge{
					From:     edge.To,
					To:       edge.From,
					Dist:     edge.Dist,
					Modes:    getBidirectionalModes(edge.Modes),
					ModeMask: edge.ModeMask & bidirectionalMask,
					Desc:     edge.Desc + " (反向)",
				}
				g.AdjList[edge.To] = append(g.AdjList[edge.To], reverseEdge)
			}
		}
	}

	return g, nil
}

// GetNeighbors 获取指定节点在特定交通方式下的邻居边
func (g *Graph) GetNeighbors(nodeID string, modeMask int) []*model.Edge {
	var validEdges []*model.Edge
	for _, edge := range g.AdjList[nodeID] {
		if edge.ModeMask&modeMask != 0 {
			validEdges = append(validEdges, edge)
		}
	}
	return validEdges
}

// FindNearestNode 找到离给定坐标最近的节点
func (g *Graph) FindNearestNode(lat, lng float64) *model.Node {
	var nearest *model.Node
	minDist := -1.0

	target := model.Point{Lat: lat, Lng: lng}
	for _, node := range g.Nodes {
		p := model.Point{Lat: node.Lat, Lng: node.Lng}
		dist := utils.HaversineDistance(target, p)

		if minDist < 0 || dist < minDist {
			minDist = dist
			nearest = node
		}
	}

	return nearest
}

// getBidirectionalModes 辅助函数：提取双向模式
func getBidirectionalModes(modes []string) []string {
	bidirectional := []string{}
	for _, m := range modes {
		if m == "walk" || m == "bike" || m == "car" {
			bidirectional = append(bidirectional, m)
		}
	}
	return bidirectional
}
