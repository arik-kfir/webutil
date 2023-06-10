package webutil

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

type HTTPConfig struct {
	Port int        `env:"PORT" value-name:"PORT" long:"port" description:"Port to listen on" default:"8000"`
	CORS CORSConfig `group:"cors" namespace:"cors" env-namespace:"CORS"`
}

type CORSConfig struct {
	AllowedOrigins     []string      `env:"ALLOWED_ORIGINS" value-name:"ORIGIN" long:"allowed-origins" description:"List of origins a cross-domain request can be executed from (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin)" required:"yes"`
	AllowMethods       []string      `env:"ALLOWED_METHODS" value-name:"METHOD" long:"allowed-methods" description:"List of HTTP methods a cross-domain request can use (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Methods)"`
	AllowHeaders       []string      `env:"ALLOWED_HEADERS" value-name:"NAME" long:"allowed-headers" description:"List HTTP headers a cross-domain request can use (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers)" default:"accept,authorization,content-type"`
	DisableCredentials bool          `env:"DISABLE_CREDENTIALS" long:"disable-credentials" description:"Disable access to credentials for JavaScript client code (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials)"`
	ExposeHeaders      []string      `env:"EXPOSE_HEADERS" long:"expose-headers" description:"List of HTTP headers to be made available to JavaScript browser code (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Expose-Headers)"`
	MaxAge             time.Duration `env:"MAX_AGE" value-name:"DURATION" long:"max-age" description:"How long results of preflights response can be cached (https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Max-Age)" default:"60s"`
}

func (c *CORSConfig) Configure(router *gin.Engine) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = c.AllowedOrigins
	corsConfig.AddAllowMethods(c.AllowMethods...)
	corsConfig.AddAllowHeaders(c.AllowHeaders...)
	corsConfig.AddExposeHeaders(c.ExposeHeaders...)
	corsConfig.AllowAllOrigins = false
	corsConfig.AllowBrowserExtensions = false
	corsConfig.AllowCredentials = !c.DisableCredentials
	corsConfig.AllowFiles = false
	corsConfig.AllowWebSockets = true
	corsConfig.AllowWildcard = true
	corsConfig.MaxAge = c.MaxAge
	corsMiddleware := cors.New(corsConfig)
	router.OPTIONS("/*path", corsMiddleware)
	router.Use(corsMiddleware)
}
