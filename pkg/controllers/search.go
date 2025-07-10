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
	return func(ctx *fiber.Ctx) error {
		// 요청 ID 생성
		reqId := uuid.New().String()
		ctx.Set("X-Req-Id", reqId)
		ctx.Set("Access-Control-Expose-Headers", "X-Req-Id")

		queries := ctx.Queries()
		var req requestDto.SearchQuery
		if err := utils.ParseAndValidate(queries, &req); err != nil {
			return utils.AppError(ctx, fiber.StatusBadRequest, err, "검증 오류")
		}

		limit, offset := utils.PaginationRequest(req.Limit, req.Offset)

		// reqId를 함께 전달
		posts, totalResults, err := searchService.SearchAnalyzedResponses(req, reqId)

		if err != nil {
			return utils.AppError(ctx, fiber.StatusInternalServerError, err, "검색 중 오류 발생")
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

		return ctx.JSON(response)
	}
}

// AnalyzeCycle은 OCR 결과를 분석하고 다음 OCR 요청 여부를 결정하는 핸들러입니다
func AnalyzeCycle(analyzerService _interface.AnalyzerService, detectorService _interface.DetectorService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		var req requestDto.AnalyzeCycleParam
		if err := ctx.BodyParser(&req); err != nil {
			return utils.AppError(ctx, fiber.StatusBadRequest, err, "JSON 파싱 실패")
		}
		// OCR 결과 처리 및 다음 OCR 요청
		response, err := analyzerService.AnalyzeCycle(req.State, req.Result)
		if err != nil {
			fmt.Printf("OCR 처리 실패: %v", err)
			return utils.AppError(ctx, fiber.StatusBadRequest, err, "OCR 처리 실패")
		}
		fmt.Printf("\n=== AnalyzeCycle Response 전체 정보 ===\n%+v\n", response)

		return ctx.JSON(response)
	}
}

func AnalyzePostByJobId(searchService _interface.SearchService, analyzerService _interface.AnalyzerService) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// OCR 결과 처리 및 다음 OCR 요청
		jobId := ctx.Params("jobId")
		job, result, err := searchService.GetJobDetail(jobId)

		if err != nil {
			fmt.Printf("OCR 처리 실패: %v", err)
			return utils.AppError(ctx, fiber.StatusNotFound, err, "OCR 처리 실패")
		}

		response, err := analyzerService.AnalyzeCycle(job, result)
		if err != nil {
			fmt.Printf("OCR 처리 실패: %v", err)
			return utils.AppError(ctx, fiber.StatusBadRequest, err, "OCR 처리 실패")
		}

		return ctx.JSON(response)
	}
}
