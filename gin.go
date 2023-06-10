package webutil

import (
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

//goland:noinspection GoUnusedExportedFunction
func InitGinPackage(devMode bool) {
	gin.DefaultWriter = log.Logger.Level(zerolog.TraceLevel)
	gin.DefaultErrorWriter = log.Logger.Level(zerolog.ErrorLevel)
	if devMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

//goland:noinspection GoUnusedExportedFunction
func NewGin() *gin.Engine {
	router := gin.New()
	router.ContextWithFallback = true
	router.MaxMultipartMemory = 8 << 20
	router.Use(requestid.New())
	router.Use(GinAccessLogMiddleware)
	return router
}
