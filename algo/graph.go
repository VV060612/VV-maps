package algo

import (
	"encoding/json"
	"fmt"
	"os"
	"traffic-system/model"
	"traffic-system/utils"
)

// Graph 图结构，用于路径规划
type Graph struct {
	Nodes    map[string]*model.Node  // 节点字典 (ID -> Node)
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

// LoadFromJSON 从 JSON 文件加载地图数据
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

	// 加载节点
	for i := range data.Nodes {
		node := &data.Nodes[i]
		g.Nodes[node.ID] = node
		g.NodeList = append(g.NodeList, *node)
	}

	// 加载边，并计算 ModeMask
	for i := range data.Edges {
		edge := &data.Edges[i]
		edge.ModeMask = model.ParseModes(edge.Modes)

		// 如果距离为 0，则自动计算
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

		// 为步行/骑行/驾车模式自动添加反向边
		// 公交和地铁是单向线路，不需要自动添加反向边
		bidirectionalMask := model.ModeWalk | model.ModeBike | model.ModeCar
		if edge.ModeMask&bidirectionalMask != 0 {
			// 检查是否已存在反向边（避免重复添加）
			reverseExists := false
			for _, existingEdge := range g.AdjList[edge.To] {
				if existingEdge.From == edge.To && existingEdge.To == edge.From {
					reverseExists = true
					break
				}
			}
			if !reverseExists {
				// 创建反向边，只保留双向模式
				reverseModeMask := edge.ModeMask & bidirectionalMask
				reverseEdge := &model.Edge{
					From:     edge.To,
					To:       edge.From,
					Dist:     edge.Dist,
					Modes:    getBidirectionalModes(edge.Modes),
					ModeMask: reverseModeMask,
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
		// 位运算判断是否支持该交通方式
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

// getBidirectionalModes 从模式列表中提取双向模式 (walk, bike, car)
func getBidirectionalModes(modes []string) []string {
	bidirectional := []string{}
	for _, m := range modes {
		if m == "walk" || m == "bike" || m == "car" {
			bidirectional = append(bidirectional, m)
		}
	}
	return bidirectional
}
