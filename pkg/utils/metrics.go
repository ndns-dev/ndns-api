package utils

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
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

	fmt.Println("메트릭 초기화 완료")
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
	fmt.Printf("메트릭 직접 기록: ndns_ocr_processing_time_seconds %.2f초\n", duration)
	OcrProcessingTime.Observe(duration)
}
