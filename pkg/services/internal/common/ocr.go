package common

import (
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

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
