package external

import (
	"time"

	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	repository "github.com/sh5080/ndns-go/pkg/repositories"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// ServerStatusService는 서버 상태를 관리하는 서비스입니다.
type ServerStatusService struct {
	serverStatusRepo *repository.ServerStatusRepository
	appName          string
	version          string
}

// 인터페이스 구현 확인
var _ _interface.ServerStatusService = (*ServerStatusService)(nil)

// NewServerStatusService는 새로운 서버 상태 서비스를 생성합니다.
func NewServerStatusService(
	serverStatusRepo *repository.ServerStatusRepository,
	appName string,
) *ServerStatusService {
	return &ServerStatusService{
		serverStatusRepo: serverStatusRepo,
		appName:          appName,
		version:          configs.AppVersion,
	}
}

// UpdateServerStatus는 현재 서버의 상태를 DynamoDB에 업데이트합니다.
func (s *ServerStatusService) UpdateServerStatus() error {
	// 서버 상태 계산 (상세 메트릭 수집)
	detailedMetrics := utils.GetDetailedServerMetrics()

	// 현재 시간
	now := time.Now()

	// 상태 정보 생성
	status := &model.ServerStatus{
		// 기본 식별 정보
		AppName:     s.appName,
		Version:     s.version,
		LastUpdated: now,
		ExpiresAt:   now.Add(60 * time.Second),

		// 서버 상태 요약
		Load:      detailedMetrics.Load,
		IsHealthy: detailedMetrics.IsHealthy,
		Capacity:  detailedMetrics.Capacity,

		// 세부 시스템 메트릭
		CpuUsage:          detailedMetrics.CpuUsage,
		MemoryUsage:       detailedMetrics.MemoryUsage,
		ResponseTime:      detailedMetrics.ResponseTime,
		RequestsPerSecond: detailedMetrics.RequestsPerSecond,
	}

	// Prometheus 메트릭 업데이트
	utils.UpdatePrometheusMetrics(s.appName, status.Load, status.IsHealthy, status.Capacity)

	// DynamoDB에 상태 저장
	if s.serverStatusRepo != nil {
		return s.serverStatusRepo.UpdateServerStatus(status)
	}

	return nil
}

// GetServerStatus는 현재 서버의 상태 정보를 반환합니다.
func (s *ServerStatusService) GetServerStatus() *model.ServerStatus {
	// 상세 메트릭 수집
	detailedMetrics := utils.GetDetailedServerMetrics()

	// 현재 시간
	now := time.Now()

	// 서버 상태 정보 생성
	status := &model.ServerStatus{
		AppName:     s.appName,
		Version:     s.version,
		LastUpdated: now,
		ExpiresAt:   now.Add(60 * time.Second),

		Load:      detailedMetrics.Load,
		IsHealthy: detailedMetrics.IsHealthy,
		Capacity:  detailedMetrics.Capacity,

		CpuUsage:          detailedMetrics.CpuUsage,
		MemoryUsage:       detailedMetrics.MemoryUsage,
		ResponseTime:      detailedMetrics.ResponseTime,
		RequestsPerSecond: detailedMetrics.RequestsPerSecond,
	}

	return status
}
