package viewport

func boundLoop(val, min, max int) int {
	if val < min {
		val = max
	}
	if val > max {
		val = min
	}

	return val
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}

	return val
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
