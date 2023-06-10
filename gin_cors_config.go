package webutil

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

func ConfigureGinCORS(
	router *gin.Engine,
	allowedOrigins []string,
	allowMethods []string,
	allowHeaders []string,
	exposeHeaders []string,
	maxAge time.Duration,
	disableCredentials bool) {

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = allowedOrigins
	corsConfig.AddAllowMethods(allowMethods...)
	corsConfig.AddAllowHeaders(allowHeaders...)
	corsConfig.AddExposeHeaders(exposeHeaders...)
	corsConfig.AllowAllOrigins = false
	corsConfig.AllowBrowserExtensions = false
	corsConfig.AllowCredentials = !disableCredentials
	corsConfig.AllowFiles = false
	corsConfig.AllowWebSockets = true
	corsConfig.AllowWildcard = true
	corsConfig.MaxAge = maxAge
	corsMiddleware := cors.New(corsConfig)
	router.OPTIONS("/*path", corsMiddleware)
	router.Use(corsMiddleware)
}
