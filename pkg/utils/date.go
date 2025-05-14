package utils

import "strconv"

// 포스트 날짜가 2025년 이후인지 확인하는 함수
func IsAfter2025(postDate string) bool {
	if len(postDate) < 4 {
		return false
	}

	yearStr := postDate[:4]
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return false
	}

	return year >= 2025
}
