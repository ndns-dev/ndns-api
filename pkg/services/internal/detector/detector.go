package detector

import (
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
)

// DetectorService는 탐지 서비스 구현체입니다
type DetectorService struct {
	_interface.Service
	queueService _interface.QueueService
}

// NewDetectorService는 새로운 탐지 서비스를 생성합니다
func NewDetectorService(queueService _interface.QueueService) _interface.DetectorService {
	return &DetectorService{
		queueService: queueService,
	}
}
