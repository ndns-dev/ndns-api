package controller

import (
	"github.com/gofiber/fiber/v2"
	service "github.com/sh5080/ndns-go/pkg/services"
	request "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	response "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// .SearchResponse는 검색 요청을 처리하는 핸들러입니다
func Search(searchService service.SearchService, sponsorDetectorService service.SponsorDetectorService) fiber.Handler {
	validate := utils.NewValidator()

	return func(c *fiber.Ctx) error {
		// 요청 파싱
		var req request.SearchQuery
		// 요청 유효성 검사
		errors := validate.Validate(req)
		if errors.HasErrors() {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": errors.Error(),
			})
		}

		// 기본값 설정
		limit, offset := utils.PaginationRequest(req.Limit, req.Offset)

		// 검색 실행
		posts, err := searchService.SearchBlogPosts(req.Query, limit, offset+1)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "검색 중 오류 발생: " + err.Error(),
			})
		}

		// 응답 생성
		response := response.Search{
			Keyword:          req.Query,
			TotalResults:     len(posts),
			SponsoredResults: 0,
			Page:             offset/limit + 1,
			ItemsPerPage:     limit,
			Posts:            posts,
		}

		return c.JSON(response)
	}
}
