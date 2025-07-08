package analyzer

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"log"

	"github.com/sh5080/ndns-go/pkg/configs"
	"github.com/sh5080/ndns-go/pkg/services/internal/detector"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// AnalyzeText는 텍스트를 분석하고 협찬 여부를 판단합니다
func (s *AnalyzerService) AnalyzeText(text string) (*responseDto.AnalyzedResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text가 비어있습니다")
	}

	trimmedText := strings.TrimSpace(text)

	// 한글 단어(2글자 이상) 포함 확인
	hangulRegex := regexp.MustCompile(`[가-힣]{2,}`)

	// 스티커 타입에 대한 특별 처리
	if !hangulRegex.MatchString(trimmedText) && len(trimmedText) < 10 {
		return &responseDto.AnalyzedResponse{
			IsSponsored: false,
		}, nil
	}

	// 협찬 여부 감지
	isSponsored, probability, indicators := detector.DetectSponsor(trimmedText, structure.SponsorTypeImage)

	return &responseDto.AnalyzedResponse{
		IsSponsored:        isSponsored,
		SponsorProbability: probability,
		SponsorIndicators:  indicators,
	}, nil
}

// AnalyzeCycle은 OCR 결과를 분석하고 다음 OCR 요청 여부를 결정합니다
func (s *AnalyzerService) AnalyzeCycle(state model.OcrQueueState, result model.OcrResult) (*responseDto.AnalyzeJobResponse, error) {
	// OCR 텍스트 분석
	analyzed, err := s.AnalyzeText(result.OcrText)
	if err != nil {
		return nil, fmt.Errorf("OCR 텍스트 분석 실패: %v", err)
	}
	analyzeJobResponse := responseDto.AnalyzeJobResponse{
		ReqId:              state.ReqId,
		JobId:              state.JobId,
		IsSponsored:        analyzed.IsSponsored,
		SponsorProbability: analyzed.SponsorProbability,
		SponsorIndicator:   analyzed.SponsorIndicators[0],
	}

	// LastSticker이거나 협찬이 발견된 경우 추가 분석 없이 결과 반환
	if state.CurrentPosition == model.OcrPositionLastSticker || analyzed.IsSponsored {
		// 라우터 서버로 분석 결과 전송
		fmt.Printf("\n=== State 정보 ===\n")
		fmt.Printf("JobId: %s\n", state.JobId)
		fmt.Printf("CurrentPosition: %v\n", state.CurrentPosition)
		fmt.Printf("RequestedAt: %v\n", state.RequestedAt)
		fmt.Printf("CrawlResult: %+v\n", state.CrawlResult)

		fmt.Printf("\n=== Analyzed 결과 ===\n")
		fmt.Printf("IsSponsored: %v\n", analyzed.IsSponsored)
		fmt.Printf("SponsorProbability: %v\n", analyzed.SponsorProbability)
		fmt.Printf("SponsorIndicators: %+v\n", analyzed.SponsorIndicators)

		fmt.Printf("\n=== OCR 결과 ===\n")
		fmt.Printf("OCR Text: %s\n", result.OcrText)

		routerUrl := configs.GetConfig().Server.RouterUrl + "/internal/analysis"
		analyzedResponse := utils.MustMarshal(analyzeJobResponse)
		resp, err := http.Post(routerUrl, "application/json", bytes.NewBuffer(analyzedResponse))
		fmt.Println("라우터 서버 전송 결과::: ", resp.Body)
		if err != nil {
			log.Printf("라우터 서버 전송 실패: %v", err)
			// 에러가 발생해도 분석 결과는 반환
		} else {
			resp.Body.Close()
		}

		return &analyzeJobResponse, nil
	}
	fmt.Printf("[%v] analyzed.IsSponsored false 이므로 다음 OCR 요청 / state.CurrentPosition: %v\n", state.JobId, state.CurrentPosition)
	// 다음 분석 위치가 있는 경우 SQS에 요청
	if err := s.detectorService.RequestNextOcr(state); err != nil {
		return nil, fmt.Errorf("다음 OCR 요청 실패: %v", err)
	}

	return &analyzeJobResponse, nil
}
