package routes

import (
	"net/http"
	"serverless-notification/domain/notification"
	"strconv"

	"github.com/gin-gonic/gin"
)

type NotificationRouteHandler struct {
	service *notification.Service
}

func NewNotificationRouteHandler(service *notification.Service) *NotificationRouteHandler {
	return &NotificationRouteHandler{service: service}
}

func (h *NotificationRouteHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/notifications", h.postNotification())
	router.GET("/notifications/:id", h.getNotificationByID())
	router.GET("/notifications", h.getNotificationsByUserID())
	// router.DELETE("/notifications/:id", h.deleteNotificationByID())
	// router.PUT("/notifications/:id", h.updateNotificationByID())

}

// POST /notifications
// Create a new notification
func (h *NotificationRouteHandler) postNotification() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req notification.CreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		notification, err := h.service.Create(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, notification)
	}
}

// GET /notifications/:id
// Get notification by ID
// Path Parameters:
// - id: string (required)
func (h *NotificationRouteHandler) getNotificationByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		notification, err := h.service.GetByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, notification)
	}
}

// GET /notifications
// Get notifications by user ID
// Query Parameters:
// - user_id: string (required)
// - limit: int (optional, default: 10)
// - next_token: string (optional) / last key from previous response
func (h *NotificationRouteHandler) getNotificationsByUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.Query("limit"))
		if err != nil {
			limit = 10
		}
		query := notification.ListQuery{
			UserID:    c.Query("user_id"),
			Limit:     limit,
			NextToken: c.Query("next_token"),
		}
		notifications, err := h.service.List(c.Request.Context(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, notifications)
	}
}
