package _interface

import (
	model "github.com/sh5080/ndns-go/pkg/types/models"
)

// ServerStatusService는 서버 상태 관리 서비스 인터페이스입니다
type ServerStatusService interface {
	GetServerStatus() *model.ServerStatus
}
