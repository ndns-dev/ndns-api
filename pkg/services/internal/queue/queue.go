package queue

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	sqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/sh5080/ndns-go/pkg/configs"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
)

type SQSImpl struct {
	client   *sqs.Client
	queueUrl string
}

// NewQueueService는 새로운 SQS 서비스를 생성합니다
func NewQueueService() _interface.QueueService {
	config := configs.GetConfig()
	// AWS SDK v2 설정
	cfg := aws.Config{
		Region: config.AWS.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			config.AWS.AccessKeyId,
			config.AWS.SecretAccessKey,
			"",
		)),
	}

	return &SQSImpl{
		client:   sqs.NewFromConfig(cfg),
		queueUrl: config.AWS.SQS.QueueUrl,
	}
}
