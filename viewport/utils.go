package viewport

// restricts val to be between min and max
// exclusive
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}

	return val
}

// like clamp, but loops back to min if val > max
// and vice-versa
func clampLoop(val, min, max int) int {
	if val < min {
		val = max
	}
	if val > max {
		val = min
	}

	return val
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// pops the last item form an array and returns it
// mutating the array
func pop[t any](list *[]t) *t {
	if len(*list) == 0 {
		return nil
	}

	last := (*list)[len(*list)-1]
	*list = (*list)[:len(*list)-1]

	return &last
}

// mutating append
func push[t any](list *[]t, item ...t) {
	*list = append(*list, item...)
}
