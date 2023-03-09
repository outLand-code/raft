package raft

func Min(a, b int) int {
	if a > b {
		return b
	} else {
		return a
	}
}

func Max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func InitIntArray(length, initValue int) []int {
	arr := make([]int, length)
	for i := 0; i < length; i++ {
		arr[i] = initValue
	}
	return arr
}
