package external

import (
	"runtime"

	"github.com/sh5080/ndns-go/pkg/configs"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// ServerMetricsService는 서버 메트릭을 관리하는 서비스입니다.
type ServerMetricsService struct {
	statusRepo *repository.ServerStatusRepository
}

// NewServerMetricsService는 새로운 서버 메트릭 서비스를 생성합니다.
func NewServerMetricsService(statusRepo *repository.ServerStatusRepository) *ServerMetricsService {
	return &ServerMetricsService{
		statusRepo: statusRepo,
	}
}

// GetServerMetrics는 현재 서버의 메트릭을 반환하고 DynamoDB에 저장합니다.
func (s *ServerMetricsService) GetServerMetrics(appURL string) (*model.ServerStatus, error) {
	// 시스템 리소스 메트릭 수집
	cpuUsage, memoryUsage := s.getSystemMetrics()

	// 요청 처리 관련 메트릭 수집
	requestsPerSecond, responseTime := utils.GetRequestMetrics()

	// 서버 부하 계산 - CPU와 메모리 사용률의 가중 평균
	load := (cpuUsage * 0.7) + (memoryUsage * 0.3)

	// 서버 건강 상태 결정
	isHealthy := true
	if cpuUsage > 0.9 || memoryUsage > 0.95 {
		isHealthy = false
	}

	// 서버 처리 용량 계산
	capacity := 1.0 - load
	if capacity < 0 {
		capacity = 0
	}

	// Prometheus 메트릭 업데이트
	utils.UpdatePrometheusMetrics(appURL, load, isHealthy, capacity)

	metrics := &model.ServerStatus{
		Load:              load,
		IsHealthy:         isHealthy,
		Capacity:          capacity,
		CpuUsage:          cpuUsage,
		MemoryUsage:       memoryUsage,
		RequestsPerSecond: requestsPerSecond,
		ResponseTime:      responseTime,
		Version:           configs.AppVersion,
	}

	// DynamoDB에 메트릭 저장
	if s.statusRepo != nil {
		if err := s.statusRepo.UpdateServerStatus(metrics); err != nil {
			return metrics, err
		}
	}

	return metrics, nil
}

// getSystemMetrics는 CPU와 메모리 사용률을 측정합니다.
func (s *ServerMetricsService) getSystemMetrics() (float64, float64) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	// CPU 사용률 계산
	cpuUsage := float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 25.0
	if cpuUsage > 100.0 {
		cpuUsage = 100.0
	}

	// 메모리 사용률 계산
	totalMem := float64(stats.Sys)
	usedMem := float64(stats.Alloc)
	memUsage := (usedMem / totalMem) * 100.0
	if memUsage > 100.0 {
		memUsage = 100.0
	}

	return cpuUsage / 100.0, memUsage / 100.0
}
