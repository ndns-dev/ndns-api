package response

import "time"

// HealthResponse는 상태 확인 요청에 대한 응답을 나타냅니다.
type HealthResponse struct {
	Status    string    `json:"status"`
	Time      time.Time `json:"time"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
	GoVersion string    `json:"goVersion"`
}
