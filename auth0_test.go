package webutil

import (
	"context"
	"github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHasScope(t *testing.T) {
	cases := []struct {
		scopes         string
		expectedScope  string
		expectedResult bool
	}{
		{"", "", false},
		{"", "a", false},
		{"", " a", false},
		{"", "a ", false},
		{"", " a ", false},
		{"", " a ", false},
		{"a", "a", true},
		{"a b", "a", true},
		{"a b ", "a", true},
		{"a  b ", "a", true},
		{" a  b ", "a", true},
		{"  a  b ", "a", true},
		{"a a", "a", true},
		{"a", "b", false},
		{"a ", "b", false},
		{" a", "b", false},
		{" a ", "b", false},
		{" a c", "b", false},
		{" a c ", "b", false},
		{" a:b ", "b", false},
	}
	for _, c := range cases {
		if HasScope(c.scopes, c.expectedScope) != c.expectedResult {
			t.Errorf("expected HasScope(%q, %q) to return %v", c.scopes, c.expectedScope, c.expectedResult)
		}
	}
}

func TestGetClaimsWithKey(t *testing.T) {
	vc1 := &validator.ValidatedClaims{}
	ctxWithClaims := context.WithValue(context.Background(), jwtmiddleware.ContextKey{}, vc1)
	if ac := GetClaims(ctxWithClaims); ac != vc1 {
		t.Errorf("expected GetClaims(ctxWithClaims) to return vc1, got %+v", ac)
	}
}

func TestGetClaimsWithBadType(t *testing.T) {
	catchPanic := func() {
		if r := recover(); r == nil {
			t.Errorf("expected GetClaims(ctxWithClaims) to panic")
		}
	}
	defer catchPanic()

	vc1 := []string{"a", "b"}
	ctxWithClaims := context.WithValue(context.Background(), jwtmiddleware.ContextKey{}, vc1)
	_ = GetClaims(ctxWithClaims)
}

func TestCreateAuth0JWTValidationGinMiddleware(t *testing.T) {
	engine := gin.New()
	engine.ContextWithFallback = true

	claimsFunc := func() validator.CustomClaims { return nil }
	engine.Use(CreateAuth0JWTValidationGinMiddleware(
		os.Getenv("TEST_AUTH0_DOMAIN"),
		[]string{os.Getenv("TEST_AUTH0_AUDIENCE")},
		validator.RS256,
		claimsFunc,
		jwtmiddleware.AuthHeaderTokenExtractor,
	))

	// Some test handler
	const responseBodyString = "Hello, World!"
	var claims *validator.ValidatedClaims
	engine.GET("/", func(c *gin.Context) {
		claims = GetClaims(c)
		c.String(200, responseBodyString)
	})

	// Start the server
	server := httptest.NewServer(engine.Handler())
	defer server.Close()

	clientReq, err := http.NewRequest(http.MethodGet, server.URL+"/", strings.NewReader("BODY_BODY"))
	if err != nil {
		t.Fatalf("Failed creating request: %+v", err)
	}
	accessToken, err := GetAccessToken(
		os.Getenv("TEST_AUTH0_DOMAIN"),
		os.Getenv("TEST_AUTH0_CLIENT_ID"),
		os.Getenv("TEST_AUTH0_CLIENT_SECRET"),
		os.Getenv("TEST_AUTH0_AUDIENCE"),
	)
	if err != nil {
		t.Fatalf("Failed getting access token: %+v", err)
	}
	clientReq.Header.Set("Authorization", "Bearer "+accessToken)

	expectedIssuer := "https://" + os.Getenv("TEST_AUTH0_DOMAIN") + "/"
	if resp, err := server.Client().Do(clientReq); err != nil {
		t.Fatalf("Failed executing request: %+v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	} else if actualResponseBody, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("Failed reading response body: %+v", err)
	} else if actualResponseBodyString := string(actualResponseBody); responseBodyString != actualResponseBodyString {
		t.Errorf("Expected response to be '%s', got '%s'", responseBodyString, actualResponseBodyString)
	} else if claims == nil {
		t.Errorf("Expected claims to be populated for request handler, got nil")
	} else if claims.RegisteredClaims.Issuer != expectedIssuer {
		t.Errorf("Expected claims issuer to be '%s', got '%s'", expectedIssuer, claims.RegisteredClaims.Issuer)
	} else if claims.RegisteredClaims.Subject != os.Getenv("TEST_AUTH0_CLIENT_ID")+"@clients" {
		t.Errorf("Expected claims issuer to be '%s', got '%s'", os.Getenv("TEST_AUTH0_CLIENT_ID"), claims.RegisteredClaims.Subject)
	} else if len(claims.RegisteredClaims.Audience) != 1 {
		t.Errorf("Expected claims audience to have %d item, got: %+v", 1, claims.RegisteredClaims.Audience)
	} else if claims.RegisteredClaims.Audience[0] != os.Getenv("TEST_AUTH0_AUDIENCE") {
		t.Errorf("Expected claims audience to be '%s', got '%s'", claims.RegisteredClaims.Audience[0], os.Getenv("TEST_AUTH0_AUDIENCE"))
	}
}
