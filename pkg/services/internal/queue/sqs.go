package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	sqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

// SendQueue는 Ocr 작업을 SQS 큐에 전송합니다
func (s *SQSImpl) SendQueue(request model.OcrQueueState) error {
	// 메시지를 JSON으로 변환
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("메시지 직렬화 실패: %v", err)
	}

	// SQS에 메시지 전송
	_, err = s.client.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    &s.queueUrl,
		MessageBody: aws.String(string(payload)),
	})

	if err != nil {
		return fmt.Errorf("SQS 메시지 전송 실패: %v", err)
	}

	return nil
}
