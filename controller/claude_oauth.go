package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// ValidateClaudeOAuthCredential live-validates a pasted Anthropic refresh_token (a real
// token refresh against platform.claude.com, also resolving account_uuid / org / email)
// and returns the resulting OAuth subscription blob for the admin to fill into the
// channel key field — then save the channel with auth_mode=oauth. Mirrors the Codex
// OAuth "complete" flow so it works for both new and existing channels. Admin only.
//
// Operational note: create/edit the channel DISABLED, fill the key, save, THEN enable —
// avoiding a window where two systems refresh the same refresh_token and race rotation.
func ValidateClaudeOAuthCredential(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
		Proxy        string `json:"proxy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	refresh := strings.TrimSpace(req.RefreshToken)
	if refresh == "" {
		common.ApiErrorMsg(c, "refresh_token 不能为空")
		return
	}

	res, err := service.RefreshClaudeOAuthTokenWithProxy(c.Request.Context(), refresh, strings.TrimSpace(req.Proxy))
	if err != nil {
		common.ApiErrorMsg(c, "刷新令牌校验失败: "+err.Error())
		return
	}
	key, err := service.NewClaudeOAuthKeyJSON(res, refresh)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "校验成功",
		"data": gin.H{
			"key":          key,
			"email":        res.Email,
			"account_uuid": res.AccountUUID,
			"expires_at":   res.ExpiresAt.Unix(),
		},
	})
}

// RefreshClaudeOAuthCredential forces an immediate token refresh for an Anthropic OAuth
// channel (serialized + exact-key CAS via the coordinator, with sibling write-back).
// Admin only. Returns a redacted summary — never the tokens.
func RefreshClaudeOAuthCredential(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiErrorMsg(c, "渠道ID格式错误")
		return
	}
	key, err := service.ForceRefreshClaudeChannel(c.Request.Context(), channelID)
	if err != nil {
		common.ApiErrorMsg(c, "刷新失败: "+err.Error())
		return
	}
	summary := key.RedactedSummary()
	summary["channel_id"] = channelID
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "刷新成功",
		"data":    summary,
	})
}
