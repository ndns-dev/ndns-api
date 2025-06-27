package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// NaverAPIClient는 네이버 API 요청을 처리하는 클라이언트입니다.
type NaverAPIClient struct {
	_interface.Service
}

// NewNaverAPIClient는 새로운 네이버 API 클라이언트를 생성합니다.
func NewNaverAPIClient(config *configs.EnvConfig) *NaverAPIClient {
	return &NaverAPIClient{
		Service: _interface.Service{
			Client: &http.Client{
				Timeout: time.Second * 10, // 10초 타임아웃
			},
			Config: config,
		},
	}
}

// SearchBlog는 네이버 블로그 검색 API를 호출하여 결과를 반환합니다.
func (c *NaverAPIClient) SearchBlog(query string, display int, start int) (*structure.NaverSearchResponse, error) {
	searchUrl := c.Config.Naver.SearchUrl

	params := url.Values{}
	params.Add("query", query)
	params.Add("display", fmt.Sprintf("%d", display))
	params.Add("start", fmt.Sprintf("%d", start))
	params.Add("sort", "sim") // 정확도순 정렬

	// 요청 Url 생성
	reqUrl := searchUrl + "?" + params.Encode()

	// HTTP 요청 생성
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %v", err)
	}

	// API 인증 헤더 추가
	req.Header.Add("X-Naver-Client-Id", c.Config.Naver.ClientId)
	req.Header.Add("X-Naver-Client-Secret", c.Config.Naver.ClientSecret)

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
	var searchResp structure.NaverSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %v", err)
	}

	return &searchResp, nil
}
