package webutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/secureworks/errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func HasScope(scopes, expectedScope string) bool {
	if scopes == "" || expectedScope == "" {
		return false
	}
	result := strings.Split(scopes, " ")
	for i := range result {
		if result[i] == expectedScope {
			return true
		}
	}
	return false
}

func GetClaims(ctx context.Context) *validator.ValidatedClaims {
	v := ctx.Value(jwtmiddleware.ContextKey{})
	if v == nil {
		return nil
	} else if claims, ok := v.(*validator.ValidatedClaims); ok {
		return claims
	} else {
		panic(fmt.Sprintf("unexpected claims type '%T' encountered: %+v", v, v))
	}
}

func GetAccessToken(auth0Domain, m2mClientID, m2mClientSecret, apiAudience string) (string, error) {
	accessTokenPayload := map[string]interface{}{
		"client_id":     m2mClientID,
		"client_secret": m2mClientSecret,
		"audience":      apiAudience,
		"grant_type":    "client_credentials",
	}
	payload, err := json.Marshal(accessTokenPayload)
	if err != nil {
		return "", errors.Chain(err, "failed creating access token request payload")
	}

	auth0ManagementURL := "https://" + auth0Domain + "/oauth/token"
	req, err := http.NewRequest("POST", auth0ManagementURL, bytes.NewReader(payload))
	if err != nil {
		return "", errors.Chain(err, "failed creating access token request")
	}
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Chain(err, "failed executing access token request")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Chain(err, "failed reading access token response")
	}

	var accessTokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &accessTokenResponse); err != nil {
		return "", errors.Chain(err, "failed unmarshalling access token response")
	}

	if v, ok := accessTokenResponse["access_token"]; !ok {
		return "", errors.Chain(err, "access token response did not provide an accesss token")
	} else if accessToken, ok := v.(string); !ok {
		return "", errors.Chain(err, "unexpected type '%T' encountered for access token in response: %+v", v, v)
	} else {
		return accessToken, nil
	}
}

func CreateAuth0JWTValidationGinMiddleware(
	auth0Domain string,
	audiences []string,
	algorithm validator.SignatureAlgorithm,
	customClaimsFunc func() validator.CustomClaims,
	tokenExtractors ...jwtmiddleware.TokenExtractor) func(c *gin.Context) {
	issuerURL, err := url.Parse("https://" + auth0Domain + "/")
	if err != nil {
		panic(fmt.Errorf("failed to parse issuer URL: %w", err))
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)
	jwtValidator, err := validator.New(
		provider.KeyFunc,
		algorithm,
		issuerURL.String(),
		audiences,
		validator.WithCustomClaims(customClaimsFunc),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		panic(fmt.Errorf("failed to set up a JWT validator: %w", err))
	}

	return func(c *gin.Context) {
		middleware := jwtmiddleware.New(
			jwtValidator.ValidateToken,
			jwtmiddleware.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
				_ = c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("failed to validate JWT: %w", err))
			}),
			jwtmiddleware.WithTokenExtractor(jwtmiddleware.MultiTokenExtractor(tokenExtractors...)),
		)

		next := func(w http.ResponseWriter, r *http.Request) {
			// The JWT middleware "CheckJWT" method will set the validated claims in the provided request context
			// Therefore, we need to make sure that our Gin context has the same request context so the claims are
			// available to the rest of the request handling code.
			origReq := c.Request
			c.Request = r
			c.Next()
			c.Request = origReq
		}
		handler := middleware.CheckJWT(http.HandlerFunc(next))
		handler.ServeHTTP(c.Writer, c.Request)
	}
}
