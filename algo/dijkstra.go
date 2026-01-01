package algo

import (
	"container/heap"
	"fmt"
	"math"
	"slices"
	"traffic-system/model"
)

// PathSegment 路径段信息
type PathSegment struct {
	FromID   string   `json:"from_id"`
	ToID     string   `json:"to_id"`
	Distance float64  `json:"distance"`
	Time     float64  `json:"time"`      // 预计时间 (秒)
	Modes    []string `json:"modes"`     // 可用的交通方式
	UsedMode string   `json:"used_mode"` // 实际使用的交通方式
	LineID   string   `json:"line_id,omitempty"`
	Desc     string   `json:"desc,omitempty"`
}

// PathResult 路径规划结果
type PathResult struct {
	Path          []string      // 节点 ID 序列
	Segments      []PathSegment // 路径段详情
	Distance      float64       // 总距离 (米)
	EstimatedTime float64       // 预计总时间 (秒)
	Found         bool          // 是否找到路径
}

// PriorityQueueItem 优先队列中的元素
type PriorityQueueItem struct {
	NodeID string
	Cost   float64 // 时间成本 (秒)
	Mode   string  // 到达该节点使用的交通方式
	LineID string  // 到达该节点使用的线路ID
	Index  int     // 在堆中的索引
}

// PriorityQueue 实现 heap.Interface 接口的优先队列
type PriorityQueue []*PriorityQueueItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Cost < pq[j].Cost
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityQueueItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.Index = -1 // 标记为已移除
	*pq = old[0 : n-1]
	return item
}

// Dijkstra 使用 Dijkstra 算法寻找最短时间路径
func (g *Graph) Dijkstra(startID, endID string, modeMask int) PathResult {
	if g.Nodes[startID] == nil || g.Nodes[endID] == nil {
		return PathResult{Found: false}
	}

	// 初始化时间成本、前驱和使用的边
	timeCost := make(map[string]float64)
	prev := make(map[string]string)
	prevEdge := make(map[string]*model.Edge)
	prevMode := make(map[string]string)   // 记录到达每个节点使用的交通方式
	prevLineID := make(map[string]string) // 记录到达每个节点使用的线路ID
	visited := make(map[string]bool)

	for id := range g.Nodes {
		timeCost[id] = math.Inf(1) // 无穷大
	}
	timeCost[startID] = 0

	// 初始化优先队列
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)
	heap.Push(&pq, &PriorityQueueItem{
		NodeID: startID,
		Cost:   0,
		Mode:   "",
		LineID: "",
	})

	// Dijkstra 主循环
	for pq.Len() > 0 {
		current := heap.Pop(&pq).(*PriorityQueueItem)
		currentID := current.NodeID

		// 如果已访问过，跳过
		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		// 如果到达终点，提前退出
		if currentID == endID {
			break
		}

		// 遍历邻居
		for _, edge := range g.GetNeighbors(currentID, modeMask) {
			neighborID := edge.To

			// 计算通过该边到达邻居的时间成本
			availableModes := model.FilterModesByMask(edge.Modes, modeMask)
			if len(availableModes) == 0 {
				continue
			}

			// 计算该边的时间成本，考虑换乘等待时间
			edgeTime, usedMode := model.EstimateSegmentTime(
				edge.Dist,
				availableModes,
				current.Mode,
				current.LineID,
				edge.LineID,
			)

			newCost := timeCost[currentID] + edgeTime

			// 如果找到更快的路径
			if newCost < timeCost[neighborID] {
				timeCost[neighborID] = newCost
				prev[neighborID] = currentID
				prevEdge[neighborID] = edge
				prevMode[neighborID] = usedMode
				prevLineID[neighborID] = edge.LineID
				heap.Push(&pq, &PriorityQueueItem{
					NodeID: neighborID,
					Cost:   newCost,
					Mode:   usedMode,
					LineID: edge.LineID,
				})
			}
		}
	}

	// 如果没有找到路径
	if timeCost[endID] == math.Inf(1) {
		return PathResult{Found: false}
	}

	// 回溯路径和边
	path := []string{}
	for at := endID; at != ""; at = prev[at] {
		path = append(path, at)
		if at == startID {
			break
		}
	}
	slices.Reverse(path)
	// 构建路径段信息
	var totalTime float64 = 0
	var totalDist float64 = 0
	segments := []PathSegment{}
	currentMode := ""
	currentLineID := ""

	for i := 0; i < len(path)-1; i++ {
		fromID := path[i]
		toID := path[i+1]
		edge := prevEdge[toID]
		if edge != nil {
			actualModes := model.FilterModesByMask(edge.Modes, modeMask)
			segTime, usedMode := model.EstimateSegmentTime(
				edge.Dist,
				actualModes,
				currentMode,
				currentLineID,
				edge.LineID,
			)
			totalTime += segTime
			totalDist += edge.Dist

			segments = append(segments, PathSegment{
				FromID:   fromID,
				ToID:     toID,
				Distance: edge.Dist,
				Time:     segTime,
				Modes:    actualModes,
				UsedMode: usedMode,
				LineID:   edge.LineID,
				Desc:     edge.Desc,
			})

			currentMode = usedMode
			currentLineID = edge.LineID
		}
	}

	return PathResult{
		Path:          path,
		Segments:      segments,
		Distance:      totalDist,
		EstimatedTime: totalTime,
		Found:         true,
	}
}

// FormatPath 格式化路径结果为可读字符串
func (g *Graph) FormatPath(result PathResult) string {
	if !result.Found {
		return "未找到路径"
	}

	output := fmt.Sprintf("总距离: %.2f 米 (%.2f 公里)\n", result.Distance, result.Distance/1000)
	output += fmt.Sprintf("预计时间: %.0f 秒 (%.1f 分钟)\n", result.EstimatedTime, result.EstimatedTime/60)
	output += "路径:\n"

	for i, nodeID := range result.Path {
		node := g.Nodes[nodeID]
		if node != nil {
			output += fmt.Sprintf("%d. %s (%s)\n", i+1, node.Name, nodeID)
		}
	}

	return output
}
