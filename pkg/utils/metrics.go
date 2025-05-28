package utils

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var (
	// CPU 사용량
	processCpuUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_cpu_usage",
			Help: "CPU usage percentage of the process",
		},
		[]string{"instance"},
	)

	// 메모리 사용량
	processMemoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_memory_usage",
			Help: "Memory usage percentage of the process",
		},
		[]string{"instance"},
	)

	// HTTP 요청 메트릭
	httpRequestsSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_server_requests_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"instance", "method", "path", "status"},
	)

	// 에러 카운터 메트릭
	errorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "application_errors_total",
			Help: "Total number of application errors",
		},
		[]string{"instance", "service", "type"},
	)

	// 서버 상태 메트릭
	serverMetrics = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "server_status",
			Help: "Server status metrics (load, health, capacity)",
		},
		[]string{"instance", "metric"},
	)

	// OCR 처리 시간 메트릭
	ocrProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ocr_processing_seconds",
			Help:    "Time spent processing OCR requests",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"instance"},
	)

	metricsInitialized bool
	initLock           sync.Mutex
)

// InitMetrics initializes all metrics
func InitMetrics() {
	initLock.Lock()
	defer initLock.Unlock()

	if metricsInitialized {
		return
	}

	// 메트릭 등록
	prometheus.MustRegister(processCpuUsage)
	prometheus.MustRegister(processMemoryUsage)
	prometheus.MustRegister(httpRequestsSeconds)
	prometheus.MustRegister(errorTotal)
	prometheus.MustRegister(serverMetrics)
	prometheus.MustRegister(ocrProcessingTime)

	metricsInitialized = true
	fmt.Println("Metrics initialized successfully")

	// 시스템 메트릭 수집 고루틴 시작
	go collectSystemMetrics()
}

// RecordRequest records HTTP request metrics
func RecordRequest(method, path string, status int, duration float64) {
	if !metricsInitialized {
		return
	}

	statusStr := strconv.Itoa(status)
	instance := GetInstanceName()
	httpRequestsSeconds.WithLabelValues(instance, method, path, statusStr).Observe(duration)
}

// RecordError records error metrics
func RecordError(service, errorType string) {
	if !metricsInitialized {
		return
	}
	instance := GetInstanceName()
	errorTotal.WithLabelValues(instance, service, errorType).Inc()
}

// GetInstanceName returns the instance name (usually APP_URL)
func GetInstanceName() string {
	// 환경변수에서 APP_URL을 가져오거나 기본값 사용
	instance := os.Getenv("APP_URL")
	if instance == "" {
		instance = "localhost"
	}
	return instance
}

// collectSystemMetrics continuously collects system metrics
func collectSystemMetrics() {
	instance := GetInstanceName()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		cpuUsage, memUsage := GetSystemMetrics()

		// CPU 사용량 업데이트 (백분율)
		processCpuUsage.WithLabelValues(instance).Set(cpuUsage * 100)

		// 메모리 사용량 업데이트 (백분율)
		processMemoryUsage.WithLabelValues(instance).Set(memUsage * 100)
	}
}

// GetSystemMetrics returns CPU and memory usage
func GetSystemMetrics() (float64, float64) {
	// CPU 사용률 측정
	cpuPercentages, err := cpu.Percent(time.Millisecond*100, false)
	if err != nil {
		fmt.Printf("CPU 사용률 측정 오류: %v\n", err)
		return 0.0, 0.0
	}

	cpuUsage := 0.0
	if len(cpuPercentages) > 0 {
		cpuUsage = cpuPercentages[0] / 100.0
	}

	// 메모리 사용률 측정
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		fmt.Printf("메모리 사용률 측정 오류: %v\n", err)
		return cpuUsage, 0.0
	}
	memoryUsage := vmStat.UsedPercent / 100.0

	return cpuUsage, memoryUsage
}

// UpdateServerMetric updates server status metrics
func UpdateServerMetric(serverName, metricName string, value float64) {
	if !metricsInitialized {
		return
	}
	serverMetrics.WithLabelValues(serverName, metricName).Set(value)
}

// RecordOcrProcessingTime records OCR processing duration
func RecordOcrProcessingTime(duration float64) {
	if !metricsInitialized {
		return
	}
	instance := GetInstanceName()
	ocrProcessingTime.WithLabelValues(instance).Observe(duration)
}
