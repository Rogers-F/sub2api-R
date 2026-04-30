package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type EnterpriseHandler struct {
	enterpriseService *service.EnterpriseService
}

func NewEnterpriseHandler(enterpriseService *service.EnterpriseService) *EnterpriseHandler {
	return &EnterpriseHandler{enterpriseService: enterpriseService}
}

type EnterpriseRequest struct {
	Name   string  `json:"name" binding:"required"`
	Notes  *string `json:"notes"`
	Status string  `json:"status" binding:"omitempty,oneof=active disabled"`
}

type EnterpriseAccountsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

func (h *EnterpriseHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filters := service.EnterpriseListFilters{
		Search: c.Query("search"),
		Status: c.Query("status"),
	}
	enterprises, total, err := h.enterpriseService.List(c.Request.Context(), page, pageSize, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*dto.Enterprise, 0, len(enterprises))
	for i := range enterprises {
		out = append(out, dto.EnterpriseFromService(&enterprises[i]))
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *EnterpriseHandler) Create(c *gin.Context) {
	var req EnterpriseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	enterprise, err := h.enterpriseService.Create(c.Request.Context(), service.CreateEnterpriseInput{
		Name:   req.Name,
		Notes:  req.Notes,
		Status: req.Status,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.EnterpriseFromService(enterprise))
}

func (h *EnterpriseHandler) GetByID(c *gin.Context) {
	enterpriseID, ok := parseEnterpriseIDParam(c)
	if !ok {
		return
	}
	enterprise, err := h.enterpriseService.Get(c.Request.Context(), enterpriseID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.EnterpriseFromService(enterprise))
}

func (h *EnterpriseHandler) Update(c *gin.Context) {
	enterpriseID, ok := parseEnterpriseIDParam(c)
	if !ok {
		return
	}
	var req EnterpriseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	enterprise, err := h.enterpriseService.Update(c.Request.Context(), enterpriseID, service.UpdateEnterpriseInput{
		Name:   req.Name,
		Notes:  req.Notes,
		Status: req.Status,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.EnterpriseFromService(enterprise))
}

func (h *EnterpriseHandler) Delete(c *gin.Context) {
	enterpriseID, ok := parseEnterpriseIDParam(c)
	if !ok {
		return
	}
	if err := h.enterpriseService.Delete(c.Request.Context(), enterpriseID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Enterprise deleted successfully"})
}

func (h *EnterpriseHandler) ListAccounts(c *gin.Context) {
	enterpriseID, ok := parseEnterpriseIDParam(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}
	groupID, ok := parseAccountGroupFilter(c)
	if !ok {
		return
	}
	accounts, total, err := h.enterpriseService.ListAccounts(
		c.Request.Context(),
		enterpriseID,
		page,
		pageSize,
		c.Query("platform"),
		c.Query("type"),
		c.Query("status"),
		search,
		groupID,
		strings.TrimSpace(c.Query("privacy_mode")),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*dto.Account, 0, len(accounts))
	for i := range accounts {
		out = append(out, dto.AccountFromService(&accounts[i]))
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *EnterpriseHandler) AssignAccounts(c *gin.Context) {
	enterpriseID, ok := parseEnterpriseIDParam(c)
	if !ok {
		return
	}
	var req EnterpriseAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	affected, err := h.enterpriseService.AssignAccounts(c.Request.Context(), enterpriseID, req.AccountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"moved": affected})
}

func (h *EnterpriseHandler) UnassignAccounts(c *gin.Context) {
	enterpriseID, ok := parseEnterpriseIDParam(c)
	if !ok {
		return
	}
	var req EnterpriseAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	affected, err := h.enterpriseService.UnassignAccounts(c.Request.Context(), enterpriseID, req.AccountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"moved": affected})
}

func parseEnterpriseIDParam(c *gin.Context) (int64, bool) {
	enterpriseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || enterpriseID <= 0 {
		response.ErrorFrom(c, service.ErrInvalidEnterpriseID)
		return 0, false
	}
	return enterpriseID, true
}

func parseAccountGroupFilter(c *gin.Context) (int64, bool) {
	groupIDStr := c.Query("group")
	if groupIDStr == "" {
		return 0, true
	}
	if groupIDStr == accountListGroupUngroupedQueryValue {
		return service.AccountListGroupUngrouped, true
	}
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil || groupID < 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_GROUP_FILTER", "invalid group filter"))
		return 0, false
	}
	return groupID, true
}
