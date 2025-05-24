package main

import (
	"github.com/sh5080/ndns-go/pkg/serverless"
)

func main() {
	// AWS Lambda 핸들러 실행
	serverless.LambdaMain()
}
