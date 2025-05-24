package utils

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

// 직접 등록할 수 있도록 메트릭을 promauto 대신 일반 prometheus로 선언
var (
	// RequestCounter는 총 요청 수를 추적합니다
	RequestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ndns_http_requests_total",
		Help: "총 HTTP 요청 수",
	}, []string{"method", "path", "status"})

	// ResponseTime은 응답 시간을 측정합니다
	ResponseTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ndns_http_response_time_seconds",
		Help:    "HTTP 요청 응답 시간(초)",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "path", "status"})

	// ApiCallCounter는 외부 API 호출 수를 추적합니다
	ApiCallCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ndns_api_calls_total",
		Help: "외부 API 호출 수",
	}, []string{"api", "status"})

	// ApiResponseTime은 외부 API 응답 시간을 측정합니다
	ApiResponseTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ndns_api_response_time_seconds",
		Help:    "외부 API 응답 시간(초)",
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10, 15, 20, 30},
	}, []string{"api"})

	// OcrProcessingTime은 OCR 처리 시간을 측정합니다
	OcrProcessingTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "ndns_ocr_processing_time_seconds",
		Help:    "OCR 처리 시간(초)",
		Buckets: []float64{0.1, 0.5, 1, 2, 3, 4, 5, 7.5, 10, 15, 20, 30},
	})

	// ErrorCounter는 오류 발생 수를 추적합니다
	ErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ndns_error_total",
		Help: "오류 발생 수",
	}, []string{"service", "type"})

	// ServerLoad는 서버 부하를 측정합니다
	ServerLoad = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ndns_server_load",
		Help: "서버 부하 (0-1)",
	}, []string{"server"})

	// ServerHealthy는 서버 정상 여부를 나타냅니다
	ServerHealthy = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ndns_server_healthy",
		Help: "서버 정상 여부 (1=정상, 0=비정상)",
	}, []string{"server"})

	// ServerCapacity는 서버 용량을 측정합니다
	ServerCapacity = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ndns_server_capacity",
		Help: "서버 용량 (0-1)",
	}, []string{"server"})
)

// 요청 메트릭 저장을 위한 변수
var (
	requestCountMutex    sync.Mutex
	requestCount         int
	lastRequestCount     int
	lastRequestCountTime time.Time
	requestsPerSecond    float64
	responseTimeSum      float64
	responseTimeCount    int
	avgResponseTime      float64
)

// InitMetrics는 모든 메트릭을 등록합니다
func InitMetrics() {
	// 모든 메트릭을 기본 레지스트리에 등록
	prometheus.MustRegister(RequestCounter)
	prometheus.MustRegister(ResponseTime)
	prometheus.MustRegister(ApiCallCounter)
	prometheus.MustRegister(ApiResponseTime)
	prometheus.MustRegister(OcrProcessingTime)
	prometheus.MustRegister(ErrorCounter)
	prometheus.MustRegister(ServerLoad)
	prometheus.MustRegister(ServerHealthy)
	prometheus.MustRegister(ServerCapacity)

	// 초기화
	lastRequestCountTime = time.Now()

	// 시스템 메트릭 수집기 시작 코드 제거 (API 호출 방식으로 대체)

	fmt.Println("메트릭 초기화 완료")
}

// RecordRequest는 HTTP 요청을 기록합니다
func RecordRequest(method, path string, status int, duration float64) {
	RequestCounter.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Inc()
	ResponseTime.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Observe(duration)

	// 요청 속도 및 응답 시간 계산을 위한 데이터 업데이트
	requestCountMutex.Lock()
	defer requestCountMutex.Unlock()

	requestCount++
	responseTimeSum += duration
	responseTimeCount++

	// 주기적으로 초당 요청 수 및 평균 응답 시간 계산
	now := time.Now()
	if now.Sub(lastRequestCountTime) >= time.Second*5 {
		elapsed := now.Sub(lastRequestCountTime).Seconds()
		if elapsed > 0 {
			requestsPerSecond = float64(requestCount-lastRequestCount) / elapsed
		}
		if responseTimeCount > 0 {
			avgResponseTime = responseTimeSum / float64(responseTimeCount)
		}

		lastRequestCount = requestCount
		lastRequestCountTime = now
		responseTimeSum = 0
		responseTimeCount = 0
	}
}

// RecordApiCall은 외부 API 호출 메트릭을 기록합니다
func RecordApiCall(apiName string, statusCode int, duration float64) {
	status := "success"
	if statusCode < 200 || statusCode >= 400 {
		status = "error"
	}
	ApiCallCounter.WithLabelValues(apiName, status).Inc()
	ApiResponseTime.WithLabelValues(apiName).Observe(duration)
}

// RecordError는 오류 발생을 기록합니다
func RecordError(service string, errorType string) {
	ErrorCounter.WithLabelValues(service, errorType).Inc()
}

// RecordOcrProcessingTime은 OCR 처리 시간을 기록합니다
func RecordOcrProcessingTime(duration float64) {
	OcrProcessingTime.Observe(duration)
}

// GetRequestMetrics는 현재의 요청 처리 메트릭을 반환합니다
func GetRequestMetrics() (float64, float64) {
	requestCountMutex.Lock()
	defer requestCountMutex.Unlock()
	return requestsPerSecond, avgResponseTime
}

// GetSystemMetrics는 시스템 메트릭(CPU, 메모리)를 수집합니다.
// gopsutil을 사용하여 정확한 CPU 및 메모리 사용률을 반환합니다.
func GetSystemMetrics() (float64, float64) {
	// 실시간으로 시스템 메트릭 측정
	// CPU 사용률 측정
	cpuPercentages, err := cpu.Percent(time.Millisecond*100, false) // 짧은 시간(100ms)으로 측정
	if err != nil {
		fmt.Printf("CPU 사용률 측정 오류: %v\n", err)
		return 0.0, 0.0 // 오류 발생 시 0.0 반환
	}

	cpuUsage := 0.0
	if len(cpuPercentages) > 0 {
		cpuUsage = cpuPercentages[0] / 100.0 // 0.0 ~ 1.0 범위로 정규화
	}

	// 메모리 사용률 측정
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		fmt.Printf("메모리 사용률 측정 오류: %v\n", err)
		return 0.0, 0.0 // 오류 발생 시 0.0 반환
	}
	memoryUsage := vmStat.UsedPercent / 100.0 // 0.0 ~ 1.0 범위로 정규화

	return cpuUsage, memoryUsage
}

// UpdateServerMetric은 서버 메트릭을 Prometheus에 업데이트합니다
func UpdateServerMetric(serverName string, metricName string, value float64) {
	switch metricName {
	case "load":
		ServerLoad.WithLabelValues(serverName).Set(value)
	case "healthy":
		ServerHealthy.WithLabelValues(serverName).Set(value)
	case "capacity":
		ServerCapacity.WithLabelValues(serverName).Set(value)
	}
}
