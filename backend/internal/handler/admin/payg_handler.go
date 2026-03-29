package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PaygHandler struct {
	paygService *service.PaygService
}

func NewPaygHandler(paygService *service.PaygService) *PaygHandler {
	return &PaygHandler{paygService: paygService}
}

func (h *PaygHandler) GetWallet(c *gin.Context) {
	wallet, err := h.paygService.GetAdminWallet(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, wallet)
}
