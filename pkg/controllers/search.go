package controller

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	_interface "github.com/sh5080/ndns-go/pkg/interfaces"
	requestDto "github.com/sh5080/ndns-go/pkg/types/dtos/requests"
	responseDto "github.com/sh5080/ndns-go/pkg/types/dtos/responses"
	"github.com/sh5080/ndns-go/pkg/utils"
)

// Search는 검색 요청을 처리하는 핸들러입니다
func Search(searchService _interface.SearchService, sponsorService _interface.SponsorService) fiber.Handler {
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

		posts, err := searchService.SearchBlogPosts(req)
		fmt.Printf("검색 결과: %+v\n", posts)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "검색 중 오류 발생: " + err.Error(),
			})
		}

		response := responseDto.Search{
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
