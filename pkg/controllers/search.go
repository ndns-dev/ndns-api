package controller

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	requestDto "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// Search는 검색 요청을 처리하는 핸들러입니다
func Search(searchService _interface.SearchService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 요청 ID 생성
		reqId := uuid.New().String()
		c.Set("X-ReqId", reqId)

		queries := c.Queries()
		var req requestDto.SearchQuery
		if err := utils.ParseAndValidate(queries, &req); err != nil {
			fmt.Printf("[ReqId: %s] 검증 오류: %v\n", reqId, err)
			return err
		}
		fmt.Printf("[ReqId: %s] 검증된 DTO: %+v\n", reqId, req)

		limit, offset := utils.PaginationRequest(req.Limit, req.Offset)
		fmt.Printf("[ReqId: %s] limit: %d, offset: %d\n", reqId, limit, offset)

		// reqId를 함께 전달
		posts, totalResults, err := searchService.SearchAnalyzedResponses(req, reqId)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "검색 중 오류 발생: " + err.Error(),
			})
		}
		var SponsoredResults int
		for _, post := range posts {
			if post.IsSponsored {
				SponsoredResults++
			}
		}

		response := responseDto.Search{
			Keyword:          req.Query,
			TotalResults:     totalResults,
			SponsoredResults: SponsoredResults,
			Page:             offset/limit + 1,
			ItemsPerPage:     limit,
			Posts:            posts,
		}

		return c.JSON(response)
	}
}

// AnalyzeText는 텍스트 분석을 요청하는 핸들러입니다
func AnalyzeText(analyzerService _interface.AnalyzerService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req requestDto.AnalyzeTextParam
		if err := c.BodyParser(&req); err != nil {
			fmt.Printf("검증 오류: %v\n", err)
			return err
		}

		post, err := analyzerService.AnalyzeText(req)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "텍스트 분석 중 오류 발생: " + err.Error(),
			})
		}
		response := responseDto.AnalyzeText{
			IsSponsored: post.IsSponsored,
			Probability: post.SponsorProbability,
			Indicators:  post.SponsorIndicators,
		}
		return c.JSON(response)
	}
}

// AnalyzeCycle은 OCR 결과를 분석하고 다음 OCR 요청 여부를 결정하는 핸들러입니다
func AnalyzeCycle(analyzerService _interface.AnalyzerService, detectorService _interface.DetectorService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 요청 바디 로깅
		bodyStr := string(c.Body())
		if bodyStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "요청 바디가 비어있습니다",
			})
		}
		fmt.Printf("수신된 요청 바디: %s\n", bodyStr)

		var req requestDto.AnalyzeCycleParam
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("JSON 파싱 실패: %v", err),
			})
		}

		// OCR 결과 처리 및 다음 OCR 요청
		response, err := analyzerService.AnalyzeCycle(req.State, req.Result)
		if err != nil {
			fmt.Printf("OCR 처리 실패: %v\n", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "OCR 처리 실패: " + err.Error(),
			})
		}

		return c.JSON(response)
	}
}
