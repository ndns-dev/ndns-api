package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	client "github.com/sh5080/ndns-go/pkg/clients"
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

	// 응답 구조체 생성
	analyzeJobResponse := responseDto.AnalyzeJobResponse{
		ReqId:              state.ReqId,
		JobId:              state.JobId,
		IsSponsored:        analyzed.IsSponsored,
		SponsorProbability: analyzed.SponsorProbability,
	}

	// SponsorIndicators가 있는 경우에만 첫 번째 인디케이터 설정
	if len(analyzed.SponsorIndicators) > 0 {
		analyzeJobResponse.SponsorIndicator = analyzed.SponsorIndicators[0]
	}

	// 바로 결과 전송하는 경우
	// 협찬이 발견된 경우
	// 마지막 이미지인 경우 (2025년 이전은 lastSticker, 2025년 이후는 secondSticker)
	isLastImage := (state.Is2025OrLater && state.CurrentPosition == model.OcrPositionSecondSticker) ||
		(!state.Is2025OrLater && state.CurrentPosition == model.OcrPositionLastSticker)

	if analyzed.IsSponsored || isLastImage {
		client.SendAnalysis(&analyzeJobResponse, &state)
		return &analyzeJobResponse, nil
	}

	// 다음 이미지 분석 시도
	if err := s.detectorService.RequestNextOcr(state); err != nil {
		if err.Error() != "finished" {
			return nil, fmt.Errorf("다음 OCR 요청 실패: %v", err)
		}
		// 더 이상 분석할 이미지가 없는 경우는 협찬이 아닌 것으로 판단
		utils.Info("AnalyzerService", "분석 완료 - 협찬 아님 (더 이상 분석할 이미지 없음): JobId=%s", state.JobId)
		analyzeJobResponse.IsSponsored = false
		analyzeJobResponse.SponsorProbability = 0
		analyzeJobResponse.SponsorIndicator = detector.CreateNonSponsoredIndicator(state.JobId)
		client.SendAnalysis(&analyzeJobResponse, &state)
		return &analyzeJobResponse, nil
	}

	return &analyzeJobResponse, nil
}
