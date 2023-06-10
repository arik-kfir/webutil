package webutil

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/secureworks/errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestGinAccessLogMiddleware(t *testing.T) {
	engine := gin.New()
	engine.ContextWithFallback = true

	// Create our own test logger that will be used by our Gin access log middleware
	accessLogBuffer := bytes.Buffer{}
	logger := zerolog.New(&accessLogBuffer)

	// Add a middleware that changes the request context's logger to our test logger
	engine.Use(func(c *gin.Context) {
		origCtx := c.Request.Context()
		newContextWithTestLogger := logger.WithContext(origCtx)
		c.Request = c.Request.WithContext(newContextWithTestLogger)
		c.Next()
		c.Request = c.Request.WithContext(origCtx)
	})

	// Add the access log middleware (which we are testing here)
	engine.Use(GinAccessLogMiddleware)

	// Some test handler
	const responseBodyString = "Hello, World!"
	engine.GET("/", func(c *gin.Context) {
		c.String(200, responseBodyString)
	})

	// Start the server
	server := httptest.NewServer(engine.Handler())
	defer server.Close()

	clientReq, err := http.NewRequest(http.MethodGet, server.URL+"/", strings.NewReader("BODY_BODY"))
	if err != nil {
		t.Fatalf("Failed creating request: %+v", err)
	}
	resp, err := server.Client().Do(clientReq)
	if err != nil {
		t.Fatalf("Failed executing request: %+v", err)
	}

	actualResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed reading response body: %+v", err)
	}

	actualResponseBodyString := string(actualResponseBody)
	if responseBodyString != actualResponseBodyString {
		t.Errorf("Expected response to be '%s', got '%s'", responseBodyString, actualResponseBodyString)
	}

	expectedAccessLogMap := map[string]interface{}{
		"http:req:header:accept-encoding": []string{"^gzip$"},
		"http:req:header:content-length":  []string{"\\d+"},
		"http:req:header:user-agent":      []string{"^Go-http-client\\/\\d+.\\d+$"},
		"http:req:host":                   ".+:\\d+",
		"http:req:method":                 "GET",
		"http:req:proto":                  "HTTP/1.1",
		"http:req:remoteAddr":             ".+:\\d+",
		"http:req:requestURI":             "^/$",
		"http:res:status":                 float64(200),
		"http:res:header:content-type":    []string{"^text\\/plain\\; charset\\=utf\\-8$"},
		"http:res:size":                   float64(13),
		"level":                           "^info$",
		"message":                         "^HTTP Request processed$",
		"request:id":                      "^$",
	}

	actualAccessLogMap := make(map[string]interface{}, 0)
	if err := json.Unmarshal(accessLogBuffer.Bytes(), &actualAccessLogMap); err != nil {
		t.Fatalf("Failed unmarshalling actual access log map: %+v", err)
	}

	if actualDuration, ok := actualAccessLogMap["http:process:duration"]; !ok {
		t.Errorf("Expected access log map to contain key 'http:process:duration', got '%+v'", actualAccessLogMap)
	} else if v, ok := actualDuration.(float64); !ok {
		t.Errorf("Expected access log map key 'http:process:duration' to be a float64, got '%+v'", actualDuration)
	} else if v < 0 {
		t.Errorf("Expected access log map key 'http:process:duration' to be > 0, got '%+v'", v)
	}
	for k, v := range expectedAccessLogMap {
		actualV, ok := actualAccessLogMap[k]
		if !ok {
			t.Errorf("Expected access log map to contain key '%s'", k)
			continue
		}

		switch expectedV := expectedAccessLogMap[k].(type) {
		case string:
			if value, ok := actualV.(string); !ok {
				t.Errorf("Expected access log entry '%s' to be a string, got '%T'", k, actualV)
			} else if matched, err := regexp.MatchString(expectedV, value); err != nil {
				t.Errorf("Error matching regular expression '%s' for access log string entry '%s': %+v", expectedV, k, errors.WithStackTrace(err))
			} else if !matched {
				t.Errorf("Expected access log string entry '%s' to be '%v', got '%+v'", k, expectedV, actualV)
			}
		case []string:
			if value, ok := actualV.([]interface{}); !ok {
				t.Errorf("Expected access log entry '%s' to be a interface{} array, got '%T'", k, actualV)
			} else if len(expectedV) != len(value) {
				t.Errorf("Expected access log string array entry '%s' to have %d items, got %d items", k, len(expectedV), len(value))
			} else {
				for i, v := range value {
					if matched, err := regexp.MatchString(expectedV[i], v.(string)); err != nil {
						t.Errorf("Error matching regular expression '%s' for access log entry '%s': %+v", expectedV[i], k, errors.WithStackTrace(err))
					} else if !matched {
						t.Errorf("Expected access log string array entry %s[%d] to be '%v', got '%+v'", k, i, expectedV[i], v)
					}
				}
			}
		case int:
			if value, ok := actualV.(int); !ok {
				t.Errorf("Expected access log int entry '%s' to be an int, got '%T'", k, actualV)
			} else if expectedV != value {
				t.Errorf("Expected access log int entry '%s' to be '%v', got '%+v'", k, expectedV, value)
			}
		case float64:
			if value, ok := actualV.(float64); !ok {
				t.Errorf("Expected access log float64 entry '%s' to be an int, got '%T'", k, actualV)
			} else if expectedV != value {
				t.Errorf("Expected access log float64 entry '%s' to be '%v', got '%+v'", k, expectedV, value)
			}
		default:
			t.Errorf("Unexpected type '%T' for access log entry '%s' with value '%v'", actualV, k, v)
		}
	}
}
