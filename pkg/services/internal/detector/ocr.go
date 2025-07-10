package detector

import (
	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// RequestNextOcr은 다음 Ocr 처리를 요청합니다
func (s *DetectorService) RequestNextOcr(state model.OcrQueueState) error {
	utils.Info("RequestNextOcr", "RequestNextOcr 시작: %+v\n", state.CurrentPosition)
	nextPosition := GetNextOcrPosition(state.CurrentPosition, state.Is2025OrLater)
	utils.Info("RequestNextOcr", "확인하는 이미지 위치: %+v\n", nextPosition)

	state.CurrentPosition = nextPosition
	return s.queueService.SendQueue(state)
}

// CreatePendingIndicator는 Ocr 분석 중임을 나타내는 지표를 생성합니다
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

// GetNextOcrPosition은 현재 위치에 따른 다음 OCR 위치를 반환합니다
func GetNextOcrPosition(current model.OcrPosition, is2025OrLater bool) model.OcrPosition {
	switch current {
	case model.OcrPositionStart:
		return model.OcrPositionFirstImage
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
