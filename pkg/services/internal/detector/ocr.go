package detector

import (
	"fmt"

	model "github.com/sh5080/ndns-go/pkg/types/models"
	structure "github.com/sh5080/ndns-go/pkg/types/structures"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// RequestNextOcr은 다음 Ocr 처리를 요청합니다
func (s *DetectorService) RequestNextOcr(state model.OcrQueueState) error {
	utils.Info("RequestNextOcr", "RequestNextOcr 시작: %+v\n", state.CurrentPosition)
	nextPosition, imageUrl := GetNextOcrPosition(state.CurrentPosition, state.CrawlResult, state.Is2025OrLater)
	utils.Info("RequestNextOcr", "확인하는 이미지 위치: %+v, URL: %s\n", nextPosition, imageUrl)

	if nextPosition == "" || imageUrl == "" {
		return fmt.Errorf("finished")
	}

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

// CreateNonSponsoredIndicator는 협찬이 아님을 나타내는 지표를 생성합니다
func CreateNonSponsoredIndicator(jobId string) structure.SponsorIndicator {
	return structure.SponsorIndicator{
		Type:        structure.IndicatorTypeKeyword,
		Pattern:     structure.PatternTypeNormal,
		MatchedText: "",
		Probability: 0,
		Source: structure.SponsorSource{
			SponsorType: structure.SponsorTypeImage,
			Text:        jobId,
		},
	}
}

// GetNextOcrPosition은 현재 위치에 따른 다음 OCR 위치와 URL을 반환합니다
func GetNextOcrPosition(current model.OcrPosition, crawlResult *structure.CrawlResult, is2025OrLater bool) (model.OcrPosition, string) {
	// 2025년 이후 포스트는 마지막 이미지와 스티커를 확인하지 않음
	positions := []struct {
		pos        model.OcrPosition
		url        string
		needsCheck bool
	}{
		{model.OcrPositionFirstImage, crawlResult.FirstImageUrl, true},
		{model.OcrPositionFirstSticker, crawlResult.FirstStickerUrl, true},
		{model.OcrPositionSecondSticker, crawlResult.SecondStickerUrl, true},
		{model.OcrPositionLastImage, crawlResult.LastImageUrl, !is2025OrLater},
		{model.OcrPositionLastSticker, crawlResult.LastStickerUrl, !is2025OrLater},
	}

	// 현재 위치가 Start면 첫 번째 유효한 위치 반환
	if current == model.OcrPositionStart {
		for _, p := range positions {
			if p.needsCheck && p.url != "" {
				return p.pos, p.url
			}
		}
		return "", ""
	}

	// 현재 위치 이후의 첫 번째 유효한 위치 반환
	foundCurrent := false
	for _, p := range positions {
		if foundCurrent {
			if p.needsCheck && p.url != "" {
				return p.pos, p.url
			}
		} else if p.pos == current {
			foundCurrent = true
		}
	}

	return "", ""
}
