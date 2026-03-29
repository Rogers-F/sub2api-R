package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PaygHandler struct {
	paygService *service.PaygService
}

func NewPaygHandler(paygService *service.PaygService) *PaygHandler {
	return &PaygHandler{paygService: paygService}
}

type PaygPrecreateRequest struct {
	Amount float64 `json:"amount"`
	Payway string  `json:"payway"`
}

type PaygCallbackRequest struct {
	SN       string `json:"sn" form:"sn"`
	ClientSN string `json:"client_sn" form:"client_sn"`
}

func (h *PaygHandler) GetWallet(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	wallet, err := h.paygService.GetWallet(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, wallet)
}

func (h *PaygHandler) Precreate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req PaygPrecreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.paygService.Precreate(c.Request.Context(), subject.UserID, req.Amount, req.Payway)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *PaygHandler) QueryOrder(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orderID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || orderID <= 0 {
		response.BadRequest(c, "Invalid order ID")
		return
	}

	order, err := h.paygService.QueryOrderForUser(c.Request.Context(), subject.UserID, orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, order)
}

func (h *PaygHandler) HandleCallback(c *gin.Context) {
	var req PaygCallbackRequest
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if _, err := h.paygService.HandleCallback(c.Request.Context(), req.SN, req.ClientSN); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.JSON(200, gin.H{"result": "success"})
}
