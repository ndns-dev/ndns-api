package _interface

import (
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

// OcrRepository는 Ocr 작업과 결과를 관리하는 인터페이스입니다
type OcrRepository interface {
	SaveOcrJob(jobDetail *model.OcrQueueState) error
	GetOcrJob(jobId string) (*model.OcrQueueState, error)
	SaveOcrResult(result *model.OcrResult) error
	GetOcrResult(imageUrl string) (*model.OcrResult, error)
}

// AnalyzedResultRepository는 분석결과를 관리하는 인터페이스입니다
type AnalyzedResultRepository interface {
	GetAnalyzedResult(link string) (*model.AnalyzedResult, error)
	SaveAnalyzedResult(result *model.AnalyzedResult) error
}
