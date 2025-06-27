package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	"github.com/sh5080/ndns-go/pkg/services/internal/common"
	requestDto "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
)

// AnalyzerService는 텍스트 분석을 위한 서비스입니다
type AnalyzerService struct {
	config *configs.EnvConfig
}

// NewAnalyzerService는 새로운 AnalyzerService를 생성합니다
func NewAnalyzerService() _interface.AnalyzerService {
	return &AnalyzerService{
		config: configs.GetConfig(),
	}
}

// AnalyzeText는 텍스트를 분석하고 협찬 여부를 판단합니다
func (s *AnalyzerService) AnalyzeText(req requestDto.AnalyzeTextParam) (*structure.AnalyzedResponse, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text가 비어있습니다")
	}

	trimmedText := strings.TrimSpace(req.Text)

	// 한글 단어(2글자 이상) 포함 확인
	hangulRegex := regexp.MustCompile(`[가-힣]{2,}`)

	// 스티커 타입에 대한 특별 처리
	if !hangulRegex.MatchString(trimmedText) && len(trimmedText) < 10 {
		return &structure.AnalyzedResponse{
			IsSponsored: false,
		}, nil
	}

	// 협찬 여부 감지
	isSponsored, probability, indicators := common.DetectSponsor(trimmedText, structure.SponsorTypeImage)

	return &structure.AnalyzedResponse{
		IsSponsored:        isSponsored,
		SponsorProbability: probability,
		SponsorIndicators:  indicators,
	}, nil
}

// AnalyzeCycle은 OCR 결과를 분석하고 다음 OCR 요청 여부를 결정합니다
func (s *AnalyzerService) AnalyzeCycle(state model.OcrQueueState, result model.OcrResult) (*structure.AnalyzedResponse, error) {
	req := requestDto.AnalyzeTextParam{
		Text: result.OcrText,
	}

	// OCR 텍스트 분석
	analyzed, err := s.AnalyzeText(req)
	if err != nil {
		return nil, fmt.Errorf("OCR 텍스트 분석 실패: %v", err)
	}

	// 협찬이 발견되지 않은 경우 다음 분석 상태 표시
	if !analyzed.IsSponsored {
		nextPosition := GetNextOcrPosition(state.CurrentPosition, state.Is2025OrLater)
		if nextPosition != "" {
			pendingIndicator := CreatePendingIndicator(state.JobId)
			analyzed.SponsorIndicators = append(analyzed.SponsorIndicators, pendingIndicator)
		}
	}

	return analyzed, nil
}

// GetNextOcrPosition은 현재 위치에 따른 다음 OCR 위치를 반환합니다
func GetNextOcrPosition(current model.OcrPosition, is2025OrLater bool) model.OcrPosition {
	switch current {
	case model.OcrPositionFirstImage:
		return model.OcrPositionFirstSticker
	case model.OcrPositionFirstSticker:
		return model.OcrPositionSecondSticker
	case model.OcrPositionSecondSticker:
		if !is2025OrLater {
			return model.OcrPositionLastImage
		}
		return ""
	case model.OcrPositionLastImage:
		if !is2025OrLater {
			return model.OcrPositionLastSticker
		}
		return ""
	case model.OcrPositionLastSticker:
		return ""
	default:
		return ""
	}
}

// CreatePendingIndicator는 OCR 분석 중임을 나타내는 지표를 생성합니다
func CreatePendingIndicator(jobId string) structure.SponsorIndicator {
	return structure.SponsorIndicator{
		Type:        structure.IndicatorTypePending,
		Pattern:     structure.PatternTypeNormal,
		MatchedText: "분석 중입니다. 잠시만 기다려주세요.",
		Probability: 0,
		Source: structure.SponsorSource{
			SponsorType: structure.SponsorTypeImage,
			Text:        jobId,
		},
	}
}
