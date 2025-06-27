package controller

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	requestDto "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	model "github.com/sh5080/ndns-go/pkg/types/models"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// Search는 검색 요청을 처리하는 핸들러입니다
func Search(searchService _interface.SearchService, postService _interface.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		queries := c.Queries()
		var req requestDto.SearchQuery
		if err := utils.ParseAndValidate(queries, &req); err != nil {
			fmt.Printf("검증 오류: %v\n", err)
			return err
		}
		fmt.Printf("검증된 DTO: %+v\n", req)

		limit, offset := utils.PaginationRequest(req.Limit, req.Offset)
		fmt.Printf("limit: %d, offset: %d\n", limit, offset)

		posts, totalResults, err := searchService.SearchAnalyzedResponses(req)

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
func AnalyzeCycle(analyzerService _interface.AnalyzerService, ocrService _interface.OcrService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			State  model.OcrQueueState `json:"state"`
			Result model.OcrResult     `json:"result"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "요청 데이터 파싱 실패: " + err.Error(),
			})
		}

		// OCR 결과 처리 및 다음 OCR 요청
		response, err := ocrService.ProcessOcrAndRequestNext(req.Result)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "OCR 처리 실패: " + err.Error(),
			})
		}

		return c.JSON(response)
	}
}
