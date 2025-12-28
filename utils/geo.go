package utils

import (
	"math"
	"traffic-system/model"
)

// EarthRadius WGS84 参考椭球长半轴 (米)
const EarthRadius = 6378137.0

// DegreesToRadians 角度转弧度
func DegreesToRadians(d float64) float64 {
	return d * math.Pi / 180.0
}

// HaversineDistance Haversine 公式 (直接计算两点间球面距离)
// 用于 Dijkstra 算法中计算 Edge 的 Weight
// 精度：高，适用于全球范围
func HaversineDistance(p1, p2 model.Point) float64 {
	lat1 := DegreesToRadians(p1.Lat)
	lon1 := DegreesToRadians(p1.Lng)
	lat2 := DegreesToRadians(p2.Lat)
	lon2 := DegreesToRadians(p2.Lng)

	dLat := lat2 - lat1
	dLon := lon2 - lon1
	// a = sin²(Δlat/2) + cos(lat1) * cos(lat2) * sin²(Δlon/2)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	// c = 2 * atan2(√a, √(1-a))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadius * c
}
