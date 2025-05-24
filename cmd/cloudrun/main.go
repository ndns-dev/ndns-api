package main

import (
	"github.com/sh5080/ndns-go/pkg/serverless"
)

func main() {
	// GCP Cloud Run 서버 실행
	serverless.CloudRunMain()
}
