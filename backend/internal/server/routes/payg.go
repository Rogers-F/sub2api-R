package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterPaygRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		payg := authenticated.Group("/user/payg")
		{
			payg.GET("/wallet", h.Payg.GetWallet)
			payg.POST("/precreate", h.Payg.Precreate)
			payg.POST("/orders/:id/query", h.Payg.QueryOrder)
		}
	}

	v1.POST("/payg/callback", h.Payg.HandleCallback)
}
