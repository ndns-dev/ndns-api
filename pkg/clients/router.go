package client

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/sh5080/ndns-go/pkg/configs"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// sendAnalysis는 분석 결과를 라우터 서버로 전송합니다
func SendAnalysis(response *responseDto.AnalyzeJobResponse, state *model.OcrQueueState) {
	routerUrl := configs.GetConfig().Server.RouterUrl + "/internal/analysis"
	analyzedResponse := utils.MustMarshal(response)
	resp, err := http.Post(routerUrl, "application/json", bytes.NewBuffer(analyzedResponse))
	if err != nil {
		log.Printf("라우터 서버 전송 실패: %v", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	utils.WebhookLog("라우터 서버 응답: %+v", string(bodyBytes))
}
