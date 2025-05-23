package model

import (
	"time"
)

const (
	TableNameServerStatus = "ServerStatus"
)

// ServerStatus는 서버의 현재 상태와 성능 지표를 나타냅니다
type ServerStatus struct {
	// =================== 기본 식별 정보 ===================
	AppName     string    `json:"app_name" dynamodbav:"AppName"`        // 서버 이름 - 기본 키(Primary Key)로 사용
	Version     string    `json:"version" dynamodbav:"version"`         // 서버 소프트웨어 버전 - 버전 관리 및 배포 확인용
	LastUpdated time.Time `json:"lastUpdated" dynamodbav:"lastUpdated"` // 마지막 업데이트 시간 - 상태 정보의 최신성 판단
	ExpiresAt   time.Time `json:"expiresAt" dynamodbav:"expiresAt"`     // DynamoDB TTL 속성 - 만료 시 자동 제거 (서버 장애 탐지용)

	// =================== 서버 상태 요약 ===================
	Load      float64 `json:"load" dynamodbav:"load"`           // 서버 부하 (0-1) - CPU와 메모리 가중 평균
	IsHealthy bool    `json:"isHealthy" dynamodbav:"isHealthy"` // 정상 작동 여부 - 기본 라우팅 결정에 사용
	Capacity  float64 `json:"capacity" dynamodbav:"capacity"`   // 처리 용량 (0-1) - 추가 요청 수용 가능성

	// =================== 시스템 성능 지표 ===================
	CpuUsage          float64 `json:"cpuUsage" dynamodbav:"cpuUsage"`                   // CPU 사용률 (0-1) - 부하 판단의 핵심 요소
	MemoryUsage       float64 `json:"memoryUsage" dynamodbav:"memoryUsage"`             // 메모리 사용률 (0-1) - 리소스 제약 감지
	ResponseTime      float64 `json:"responseTime" dynamodbav:"responseTime"`           // 평균 응답 시간 (초) - 지연 시간 모니터링
	RequestsPerSecond float64 `json:"requestsPerSecond" dynamodbav:"requestsPerSecond"` // 초당 요청 수 - 현재 트래픽 측정
}
