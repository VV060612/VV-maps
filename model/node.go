package model

// Point 代表一个经纬度点 (WGS84)
type Point struct {
	Lat float64 // 纬度
	Lng float64 // 经度
}

// PointXY 代表平面坐标系中的一个点
type PointXY struct {
	X float64 // 东西向距离 (米)
	Y float64 // 南北向距离 (米)
}

// Node 对应地图上的一个点 (站点、路口、地标)
type Node struct {
	ID   string  `json:"id" gorm:"primaryKey"`
	Name string  `json:"name" gorm:"index"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Type string  `json:"type" gorm:"index"` // 如: "landmark", "subway_entrance", "bus_stop"
}
