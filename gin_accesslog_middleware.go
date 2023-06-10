package webutil

import (
	"bytes"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"strings"
	"time"
)

type customReadCloser struct {
	r io.Reader
}

func (rc *customReadCloser) Read(p []byte) (int, error) {
	return rc.r.Read(p)
}

func (rc *customReadCloser) Close() error {
	if closer, ok := rc.r.(io.Closer); ok {
		return closer.Close()
	} else {
		return nil
	}
}

func GinAccessLogMiddleware(c *gin.Context) {
	// Create the logger event which we will start adding request & response data to
	event := log.Ctx(c.Request.Context()).With()

	// Add common request data
	event = event.
		Str("request:id", requestid.Get(c)).
		Str("http:req:host", c.Request.Host).
		Str("http:req:method", c.Request.Method).
		Str("http:req:proto", c.Request.Proto).
		Str("http:req:remoteAddr", c.Request.RemoteAddr).
		Str("http:req:requestURI", c.Request.RequestURI)

	// Add transfer encoding
	if len(c.Request.TransferEncoding) > 0 {
		transferEncoding := zerolog.Arr()
		for _, encoding := range c.Request.TransferEncoding {
			transferEncoding = transferEncoding.Str(encoding)
		}
		event = event.Array("http:req:transferEncoding", transferEncoding)
	}

	// Add headers (excluding some)
	for name, values := range c.Request.Header {
		name = strings.ToLower(name)
		if strings.HasPrefix(name, "sec-") {
			continue
		}
		arr := zerolog.Arr()
		for _, value := range values {
			arr.Str(value)
		}
		event = event.Array("http:req:header:"+name, arr)
	}

	// Add trailer headers (excluding some)
	for name, values := range c.Request.Trailer {
		name = strings.ToLower(name)
		if strings.HasPrefix(name, "sec-") {
			continue
		}
		arr := zerolog.Arr()
		for _, value := range values {
			arr.Str(value)
		}
		event = event.Array("http:req:trailer:"+name, arr)
	}

	// Keep a copy of the request body
	requestBody := bytes.Buffer{}
	c.Request.Body = &customReadCloser{r: io.TeeReader(c.Request.Body, &requestBody)}

	// Replace the request context with a context that references our logger (and revert immediately after)
	origCtx := c.Request.Context()
	newContextWithReqLogger := event.Logger().WithContext(origCtx)
	c.Request = c.Request.WithContext(newContextWithReqLogger)

	// Invoke & time the next handler
	start := time.Now()
	c.Next()
	duration := time.Since(start)

	// Restore request context
	c.Request = c.Request.WithContext(origCtx)

	// If this is a health-check request, stop here
	if c.Request.RequestURI == "/healthz" {
		return
	}

	// Add invocation result
	event = event.Dur("http:process:duration", duration)
	event = event.Int("http:res:status", c.Writer.Status())
	event = event.Int("http:res:size", c.Writer.Size())

	// Add response headers
	for name, values := range c.Writer.Header() {
		if strings.HasPrefix(name, "sec-") {
			continue
		}
		arr := zerolog.Arr()
		for _, value := range values {
			arr.Str(value)
		}
		event = event.Array("http:res:header:"+strings.ToLower(name), arr)
	}

	// Add response errors
	if len(c.Errors) > 0 {
		var errorsArr []error
		for _, err := range c.Errors {
			errorsArr = append(errorsArr, err.Err)
		}
		event = event.Stack().Err(errorsArr[0])
		if len(errorsArr) > 1 {
			event = event.Errs("http:res:errors", errorsArr)
		}
	}

	// Perform the logging with all the information we've added so far
	const message = "HTTP Request processed"
	logger := &([]zerolog.Logger{event.Logger()}[0])
	if len(c.Errors) == 0 {
		if c.Writer.Status() >= 200 && c.Writer.Status() <= 399 {
			logger.Info().Msg(message)
		} else if c.Writer.Status() >= 400 && c.Writer.Status() <= 499 {
			logger.Warn().Msg(message)
		} else {
			logger.Error().Msg(message)
		}
	} else {
		logger.Error().Msg(message)
	}
}
