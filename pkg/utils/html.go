package utils

import (
	"regexp"
	"strings"
)

// RemoveHTMLTags는 문자열에서 HTML 태그를 제거합니다
func RemoveHTMLTags(s string) string {
	// HTML 태그 정규식
	re := regexp.MustCompile(`<[^>]*>`)
	noTags := re.ReplaceAllString(s, "")

	// HTML 엔티티 처리
	noTags = strings.ReplaceAll(noTags, "&lt;", "<")
	noTags = strings.ReplaceAll(noTags, "&gt;", ">")
	noTags = strings.ReplaceAll(noTags, "&amp;", "&")
	noTags = strings.ReplaceAll(noTags, "&quot;", "\"")
	noTags = strings.ReplaceAll(noTags, "&#39;", "'")

	// 여러 공백 정리
	reSpace := regexp.MustCompile(`\s+`)
	noTags = reSpace.ReplaceAllString(noTags, " ")

	return strings.TrimSpace(noTags)
}
