package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/an1shka/go-project-playerook-kz/handlers"
	"github.com/an1shka/go-project-playerook-kz/middleware"
)

func Register(router *gin.Engine, handler *handlers.Handler) {
	router.Static("/static", "./static")
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	router.GET("/health", handler.Health)
	router.POST("/api/auth/register", handler.Register)
	router.POST("/api/auth/login", handler.Login)
	router.GET("/api/listings", handler.ListListings)
	router.GET("/api/listings/:id", handler.GetListing)
	router.GET("/api/users/:username", handler.GetPublicProfile)
	router.GET("/api/news", handler.ListNews)

	protected := router.Group("/api")
	protected.Use(middleware.Auth(handler.JWTSecret))
	protected.GET("/profile", handler.Profile)
	protected.GET("/profile/listings", handler.MyListings)
	protected.POST("/listings", handler.CreateListing)
	protected.PUT("/listings/:id", handler.UpdateListing)
	protected.PATCH("/listings/:id", handler.UpdateListing)
	protected.DELETE("/listings/:id", handler.DeleteListing)
	protected.POST("/news", handler.CreateNews)

	router.GET("/ws/chat", middleware.Auth(handler.JWTSecret), handler.Chat)
}
