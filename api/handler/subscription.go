package handler

import (
"github.com/gin-gonic/gin"
"github.com/yourname/github-sentinel/internal/service"
)

type SubscriptionHandler struct {
svc service.SubscriptionService
}

func (h *SubscriptionHandler) Subscribe(c *gin.Context) {
c.JSON(200, gin.H{"message": "ok"})
}
