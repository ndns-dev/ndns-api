package _interface

import structure "github.com/sh5080/ndns-go/pkg/types/structures"

// OCRService는 이미지에서 텍스트를 추출하는 인터페이스입니다
type OCRService interface {
	// ExtractTextFromImage는 이미지 URL에서 텍스트를 추출합니다
	ExtractTextFromImage(imageURL string) (string, error)
}

type OCRRepository interface {
	// GetOCRCache는 이미지 URL에 대한 OCR 캐시를 가져옵니다
	GetOCRCache(imageURL string) (*structure.OCRCache, error)

	// SaveOCRCache는 이미지 URL에 대한 OCR 결과를 저장합니다
	SaveOCRCache(imageURL string, textDetected string, imageType string) error
}

type OCRFunc func(imageURL string) (string, error)
type OCRCacheFunc func(imageURL string) (string, bool)
