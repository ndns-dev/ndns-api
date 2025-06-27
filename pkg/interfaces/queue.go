package _interface

import model "github.com/sh5080/ndns-go/pkg/types/models"

// QueueService는 SQS 작업을 처리하는 인터페이스입니다
type QueueService interface {
	// SendQueue는 Ocr 작업을 SQS 큐에 전송합니다
	SendQueue(queueState model.OcrQueueState) error
}
