package utils

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// 로그 레벨 정의
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// 로그 레벨을 문자열로 변환
func (l LogLevel) String() string {
	return [...]string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}[l]
}

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

// LogMessage는 지정된 레벨에 해당하는 로그 메시지를 출력합니다
func LogMessage(level LogLevel, service string, format string, args ...interface{}) {
	// 디버그 모드가 아닐 때 DEBUG 로그는 출력하지 않음
	if level == DEBUG && !IsDebug() {
		return
	}

	// 호출 위치 정보 가져오기
	_, file, line, _ := runtime.Caller(1)
	// 전체 경로에서 파일명만 추출
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			file = file[i+1:]
			break
		}
	}

	// 현재 시간 포맷팅
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// 로그 메시지 생성 및 출력
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] %s [%s] %s:%d - %s",
		timestamp, level.String(), service, file, line, message)

	// 로그 레벨에 따라 출력 대상 결정
	if level >= ERROR {
		// 에러 레벨 이상은 표준 에러로 출력하고 메트릭에 기록
		fmt.Fprintln(os.Stderr, logLine)
		RecordError(service, level.String())
	} else {
		// 일반 로그는 표준 출력으로 출력
		fmt.Fprintln(os.Stdout, logLine)
	}
}

// OCRErrorLog는 OCR 처리 중 발생한 오류를 파일에 기록합니다
func OCRErrorLog(errorType string, imageURL string, errorDetail string) {
	// 현재 시간 포맷팅
	timestamp := time.Now().Format(time.RFC3339)

	// OCR 오류 로그 포맷 구성
	ocrErrorData := fmt.Sprintf("OCR_ERROR|%s|%s|%s|%s", timestamp, imageURL, errorType, errorDetail)

	// 로그 파일 경로
	logFilePath := "logs/ocr_errors.log"

	// 디렉토리 확인 및 생성
	err := os.MkdirAll("logs", 0755)
	if err != nil {
		Error("system", "OCR 로그 디렉토리 생성 실패: %v", err)
		return
	}

	// 파일 열기 (없으면 생성, 있으면 추가)
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Error("system", "OCR 로그 파일 열기 실패: %v", err)
		return
	}
	defer f.Close()

	// 로그 데이터 쓰기
	if _, err := f.WriteString(ocrErrorData + "\n"); err != nil {
		Error("system", "OCR 로그 쓰기 실패: %v", err)
	}

	// 에러 메트릭 기록
	RecordError("image_ocr", errorType)
}

// 편의성 함수들
func Debug(service, format string, args ...interface{}) {
	LogMessage(DEBUG, service, format, args...)
}

func Info(service, format string, args ...interface{}) {
	LogMessage(INFO, service, format, args...)
}

func Warn(service, format string, args ...interface{}) {
	LogMessage(WARN, service, format, args...)
}

func Error(service, format string, args ...interface{}) {
	LogMessage(ERROR, service, format, args...)
}

func Fatal(service, format string, args ...interface{}) {
	LogMessage(FATAL, service, format, args...)
	os.Exit(1)
}

// DebugLog는 디버그 모드일 때만 로그 메시지를 출력합니다 (하위 호환성 유지)
func DebugLog(format string, args ...interface{}) {
	if IsDebug() {
		Debug("system", format, args...)
	}
}
