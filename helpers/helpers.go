package helpers

import "time"

func DurationSliceMean(inp []time.Duration, unit time.Duration) float64 {
	if len(inp) > 0 {
		var totalDuration time.Duration
		for _, cDuration := range inp {
			totalDuration += cDuration
		}
		return float64(totalDuration) / float64(len(inp)) / float64(unit)
	}
	return 0
}
