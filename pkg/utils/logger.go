package utils

import (
	"fmt"
	"os"
	"sync"
)

// 디버그 모드 상태를 저장할 변수와 초기화를 한 번만 수행하기 위한 once
var isDebugMode bool
var debugOnce sync.Once

// IsDebug는 현재 애플리케이션이 디버그 모드로 실행 중인지 확인합니다
func IsDebug() bool {
	debugOnce.Do(func() {
		env := os.Getenv("APP_ENV")
		isDebugMode = env == "dev" || env == "local"
	})
	return isDebugMode
}

// DebugLog는 디버그 모드일 때만 로그 메시지를 출력합니다
func DebugLog(format string, args ...interface{}) {
	if IsDebug() {
		fmt.Printf(format, args...)
	}
}
