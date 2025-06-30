package utils

import "encoding/json"

// SliceToJSON 将字符串切片转换为JSON字符串
func SliceToJSON(slice []string) string {
	if len(slice) == 0 {
		return "[]"
	}
	jsonBytes, _ := json.Marshal(slice)
	return string(jsonBytes)
}

// JSONToSlice 将JSON字符串转换为字符串切片
func JSONToSlice(jsonStr string) []string {
	if jsonStr == "" {
		return []string{}
	}
	var slice []string
	json.Unmarshal([]byte(jsonStr), &slice)
	return slice
}
