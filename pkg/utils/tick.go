package utils

import (
	"math"
)

func FloorToTick(val, tick float64) float64 {
	if tick == 0.0 {
		return val
	}
	return math.Floor((val/tick)+0.000000001) * tick
}

func IsValidTick(val, tick float64) bool {
	if tick == 0.0 {
		return true
	}
	normalized := FloorToTick(val, tick)
	return math.Abs(val-normalized) < 0.0000000001
}

func PriceToTicks(price, tick float64) int64 {
	if tick == 0.0 {
		return 0
	}
	return int64(math.Round(price / tick))
}

func TicksToPrice(ticks int64, tick float64) float64 {
	return float64(ticks) * tick
}

func RoundToTick(val, tick float64) float64 {
	if tick == 0.0 {
		return val
	}
	return math.Round(val/tick) * tick
}
