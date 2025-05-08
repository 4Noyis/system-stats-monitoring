package stats

import "math/rand"

func GetCpuUsage() float64 {
	return rand.Float64() * 100
}
