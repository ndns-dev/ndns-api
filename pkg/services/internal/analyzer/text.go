package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sh5080/ndns-go/pkg/services/internal/detector"
	requestDto "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// AnalyzeText는 텍스트를 분석하고 협찬 여부를 판단합니다
func (s *AnalyzerService) AnalyzeText(req requestDto.AnalyzeTextParam) (*responseDto.AnalyzedResponse, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text가 비어있습니다")
	}

	trimmedText := strings.TrimSpace(req.Text)

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
func (s *AnalyzerService) AnalyzeCycle(state model.OcrQueueState, result model.OcrResult) (*responseDto.AnalyzedResponse, error) {
	req := requestDto.AnalyzeTextParam{
		Text: result.OcrText,
	}

	// OCR 텍스트 분석
	analyzed, err := s.AnalyzeText(req)
	if err != nil {
		return nil, fmt.Errorf("OCR 텍스트 분석 실패: %v", err)
	}

	// LastSticker이거나 협찬이 발견된 경우 추가 분석 없이 결과 반환
	if state.CurrentPosition == model.OcrPositionLastSticker || analyzed.IsSponsored {
		return analyzed, nil
	}

	// 다음 분석 위치가 있는 경우 SQS에 요청
	if err := s.detectorService.RequestNextOcr(state); err != nil {
		return nil, fmt.Errorf("다음 OCR 요청 실패: %v", err)
	}

	return analyzed, nil
}
