package model

// Edge 对应两点之间的一条连线
type Edge struct {
	From   string   `json:"from"`
	To     string   `json:"to"`
	Dist   float64  `json:"dist"`              // 距离 (米), 已经算好
	Modes  []string `json:"modes"`             // 原始模式列表: ["car", "bus"]
	LineID string   `json:"line_id,omitempty"` // 线路ID, 仅公交/地铁有
	Desc   string   `json:"desc,omitempty"`    // 描述

	// --- 下面这个字段 JSON 里没有，是我们在加载数据后算出来的 ---
	ModeMask int `json:"-"` // 位掩码，用于算法中毫秒级判断通行权限
}

// MapData 用于解析整个 JSON 文件
type MapData struct {
	Meta  map[string]interface{} `json:"meta"` // 存版本号等元数据
	Nodes []Node                 `json:"nodes"`
	Edges []Edge                 `json:"edges"`
}

// 定义通行模式的二进制位 (Bitmask)
// 这样做的好处：判断能不能走，不用对比字符串，只需要做一次位与运算 (&)
const (
	ModeNone   = 0
	ModeWalk   = 1 << 0 // 1  (二进制 00001)
	ModeBike   = 1 << 1 // 2  (二进制 00010)
	ModeCar    = 1 << 2 // 4  (二进制 00100)
	ModeBus    = 1 << 3 // 8  (二进制 01000)
	ModeSubway = 1 << 4 // 16 (二进制 10000)
)

// 各交通方式的平均速度 (米/秒)
const (
	SpeedWalk   = 1.4  // 步行: 约 5 km/h
	SpeedBike   = 4.2  // 骑行: 约 15 km/h
	SpeedCar    = 8.3  // 驾车: 约 30 km/h (城市道路)
	SpeedBus    = 5.5  // 公交: 约 20 km/h (含停靠)
	SpeedSubway = 10.0 // 地铁: 约 36 km/h (含停靠)
)

// 各交通方式的平均等待/准备时间 (秒)
// 这些时间反映了实际生活中的额外开销
const (
	WaitTimeWalk   = 0    // 步行: 无需等待
	WaitTimeBike   = 30   // 骑行: 找车、解锁等 (共享单车约30秒)
	WaitTimeCar    = 60   // 驾车: 找车位、启动等 (约1分钟)
	WaitTimeBus    = 300  // 公交: 平均等待时间 (约5分钟，假设10分钟一班)
	WaitTimeSubway = 180  // 地铁: 平均等待时间 (约3分钟，假设6分钟一班)
)

// ParseModes 将字符串数组转换为位掩码
// 例如: ["walk", "bike"] -> 1 | 2 = 3
func ParseModes(modes []string) int {
	mask := 0
	for _, m := range modes {
		switch m {
		case "walk":
			mask |= ModeWalk
		case "bike":
			mask |= ModeBike
		case "car":
			mask |= ModeCar
		case "bus":
			mask |= ModeBus
		case "subway":
			mask |= ModeSubway
		}
	}
	return mask
}

// GetModeSpeed 获取指定交通方式的速度 (米/秒)
func GetModeSpeed(mode string) float64 {
	switch mode {
	case "walk":
		return SpeedWalk
	case "bike":
		return SpeedBike
	case "car":
		return SpeedCar
	case "bus":
		return SpeedBus
	case "subway":
		return SpeedSubway
	default:
		return SpeedWalk // 默认步行速度
	}
}

// GetModeWaitTime 获取指定交通方式的等待/准备时间 (秒)
func GetModeWaitTime(mode string) float64 {
	switch mode {
	case "walk":
		return WaitTimeWalk
	case "bike":
		return WaitTimeBike
	case "car":
		return WaitTimeCar
	case "bus":
		return WaitTimeBus
	case "subway":
		return WaitTimeSubway
	default:
		return 0
	}
}

// GetModeMask 获取单个交通方式的位掩码
func GetModeMask(mode string) int {
	switch mode {
	case "walk":
		return ModeWalk
	case "bike":
		return ModeBike
	case "car":
		return ModeCar
	case "bus":
		return ModeBus
	case "subway":
		return ModeSubway
	default:
		return 0
	}
}

// FilterModesByMask 根据用户选择的 modeMask 过滤边支持的交通方式
// 返回用户可以实际使用的交通方式列表
func FilterModesByMask(edgeModes []string, userModeMask int) []string {
	var filtered []string
	for _, mode := range edgeModes {
		modeMask := GetModeMask(mode)
		if modeMask&userModeMask != 0 {
			filtered = append(filtered, mode)
		}
	}
	return filtered
}

// EstimateTime 根据距离和交通方式估算行驶时间 (秒)
// 注意: 此函数只计算行驶时间，不含等待时间
// 如果有多种交通方式，选择最快的
func EstimateTime(distance float64, modes []string) float64 {
	if len(modes) == 0 {
		return distance / SpeedWalk
	}

	// 找到最快的交通方式
	maxSpeed := 0.0
	for _, mode := range modes {
		speed := GetModeSpeed(mode)
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}

	if maxSpeed == 0 {
		maxSpeed = SpeedWalk
	}

	return distance / maxSpeed
}

// EstimateSegmentTime 估算路段时间，考虑实际使用的交通方式和换乘等待
// 参数:
//   - distance: 路段距离 (米)
//   - availableModes: 用户可以使用的交通方式 (已过滤)
//   - prevMode: 上一段使用的交通方式 (用于判断是否换乘，空字符串表示第一段)
//   - prevLineID: 上一段的线路ID (用于判断公交/地铁是否同线)
//   - currentLineID: 当前段的线路ID
//
// 返回:
//   - time: 预计时间 (秒)
//   - usedMode: 实际使用的交通方式
func EstimateSegmentTime(distance float64, availableModes []string, prevMode string, prevLineID string, currentLineID string) (time float64, usedMode string) {
	if len(availableModes) == 0 {
		return distance / SpeedWalk, "walk"
	}

	// 计算每种交通方式的总时间 (行驶时间 + 可能的等待时间)
	// 选择总时间最短的
	bestTime := -1.0
	bestMode := ""

	for _, mode := range availableModes {
		speed := GetModeSpeed(mode)
		travelTime := distance / speed

		// 计算等待时间
		waitTime := 0.0
		needWait := false

		switch mode {
		case "walk":
			// 步行不需要等待
			needWait = false
		case "bike", "car":
			// 骑行/驾车: 只有第一次使用或换乘时才需要准备时间
			if prevMode != mode {
				needWait = true
			}
		case "bus", "subway":
			// 公交/地铁: 换乘不同线路时需要等待
			// 如果是同一条线路的连续站点，不需要重新等待
			if prevMode != mode || (prevLineID != currentLineID && currentLineID != "") {
				needWait = true
			}
		}

		if needWait {
			waitTime = GetModeWaitTime(mode)
		}

		totalTime := travelTime + waitTime

		// 选择总时间最短的交通方式
		if bestTime < 0 || totalTime < bestTime {
			bestTime = totalTime
			bestMode = mode
		}
	}

	return bestTime, bestMode
}
