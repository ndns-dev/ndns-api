package serverless

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"
)

var fiberLambda *fiberadapter.FiberLambda

// Handler는 AWS Lambda 핸들러 함수입니다
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 첫 요청 시 Fiber 앱을 Lambda 어댑터에 연결
	if fiberLambda == nil {
		fmt.Println("AWS Lambda에서 Fiber 앱 초기화")
		fiberLambda = fiberadapter.New(GetApp())
	}

	// Lambda 요청을 Fiber 앱으로 전달
	return fiberLambda.ProxyWithContext(ctx, req)
}

// LambdaMain은 AWS Lambda 진입점 함수입니다
func LambdaMain() {
	lambda.Start(Handler)
}
