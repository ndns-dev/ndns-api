package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// NgrokAPIClient는 ngrok API 요청을 처리하는 클라이언트입니다.
type NgrokAPIClient struct {
	_interface.Service
	BaseURL string
}

// NewNgrokAPIClient는 새로운 ngrok API 클라이언트를 생성합니다.
func NewNgrokAPIClient(config *configs.EnvConfig) *NgrokAPIClient {

	ngrokAPIURL := "http://localhost:4040"
	// Docker 환경인지 확인
	if _, err := os.Stat("/.dockerenv"); err == nil {
		// Docker 컨테이너 내부
		ngrokAPIURL = "http://host.docker.internal:4040"
	}

	return &NgrokAPIClient{
		Service: _interface.Service{
			Client: &http.Client{
				Timeout: time.Second * 5, // 5초 타임아웃
			},
			Config: config,
		},
		BaseURL: ngrokAPIURL,
	}
}

// GetTunnels는 현재 활성화된 ngrok 터널 정보를 가져옵니다.
func (c *NgrokAPIClient) GetTunnels() (*structure.NgrokTunnelsResponse, error) {
	reqURL := c.BaseURL + "/api/tunnels"

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %v", err)
	}

	// 요청 실행
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %v", err)
	}

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 오류 (%d): %s", resp.StatusCode, string(body))
	}

	// 응답 JSON 파싱
	var tunnelsResp structure.NgrokTunnelsResponse
	if err := json.Unmarshal(body, &tunnelsResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}

	return &tunnelsResp, nil
}
