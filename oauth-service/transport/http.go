package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/zipkin"
	"github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/lixichongAAA/seckill/oauth-service/endpoint"
	"github.com/lixichongAAA/seckill/oauth-service/service"
	gozipkin "github.com/openzipkin/zipkin-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ErrorBadRequest         = errors.New("invalid request parameter")
	ErrorGrantTypeRequest   = errors.New("invalid request grant type")
	ErrorTokenRequest       = errors.New("invalid request token")
	ErrInvalidClientRequest = errors.New("invalid client message")
)

// MakeHttpHandler make http handler use mux
func MakeHttpHandler(ctx context.Context, endpoints endpoint.OAuth2Endpoints, tokenService service.TokenService, clientService service.ClientDetailsService, zipkinTracer *gozipkin.Tracer, logger log.Logger) http.Handler {
	r := mux.NewRouter()
	zipkinServer := zipkin.HTTPServerTrace(zipkinTracer, zipkin.Name("http-transport"))

	options := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder(encodeError),
		zipkinServer,
	}
	r.Path("/metrics").Handler(promhttp.Handler())

	clientAuthorizationOptions := []kithttp.ServerOption{
		kithttp.ServerBefore(makeClientAuthorizationContext(clientService, logger)),
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder(encodeError),
		zipkinServer,
	}
	// /oauth/token 端点用于请求访问令牌，它通过请求参数中的 grant_type 来识别请求访问令牌的授权类型
	// 并验证请求中携带的客户端凭证和用户凭证是否有效，只有通过验证的客户端请求才能获取访问令牌
	r.Methods("POST").Path("/oauth/token").Handler(kithttp.NewServer(
		endpoints.TokenEndpoint,
		decodeTokenRequest,
		encodeJsonResponse,
		clientAuthorizationOptions...,
	))
	// /oauth/check_token 端点提供给客户端和资源服务器验证访问令牌的有效性
	// 若有效，则返回访问令牌绑定的用户信息和客户端信息
	r.Methods("POST").Path("/oauth/check_token").Handler(kithttp.NewServer(
		endpoints.CheckTokenEndpoint,
		decodeCheckTokenRequest,
		encodeJsonResponse,
		clientAuthorizationOptions...,
	))

	// create health check handler
	r.Methods("GET").Path("/health").Handler(kithttp.NewServer(
		endpoints.HealthCheckEndpoint,
		decodeHealthCheckRequest,
		encodeJsonResponse,
		options...,
	))

	return r
}

// 验证 Authorization 中携带的客户端信息
// 一般来说，客户端信息会通过 base64 的方式加密后放入到 Authorization 请求头中，
// 加密前的信息按照 clientId:clientSecret
func makeClientAuthorizationContext(clientDetailsService service.ClientDetailsService, logger log.Logger) kithttp.RequestFunc {

	return func(ctx context.Context, r *http.Request) context.Context {
		// 获取 Authorization 中的客户端信息
		if clientId, clientSecret, ok := r.BasicAuth(); ok {
			// 验证客户端信息
			clientDetails, err := clientDetailsService.GetClientDetailByClientId(ctx, clientId, clientSecret)
			if err == nil {
				// 验证成功，在请求上下文放入客户端信息
				return context.WithValue(ctx, endpoint.OAuth2ClientDetailsKey, clientDetails)
			}
		}
		// 验证失败，在请求上下文放入验证失败错误信息
		return context.WithValue(ctx, endpoint.OAuth2ErrorKey, ErrInvalidClientRequest)
	}
}

func decodeTokenRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	grantType := r.URL.Query().Get("grant_type")
	if grantType == "" {
		return nil, ErrorGrantTypeRequest
	}
	return &endpoint.TokenRequest{
		GrantType: grantType,
		Reader:    r,
	}, nil

}

func decodeCheckTokenRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	tokenValue := r.URL.Query().Get("token")
	if tokenValue == "" {
		return nil, ErrorTokenRequest
	}

	return &endpoint.CheckTokenRequest{
		Token: tokenValue,
	}, nil

}

// encode errors from business-logic
func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func encodeJsonResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// decodeHealthCheckRequest decode request
func decodeHealthCheckRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return endpoint.HealthRequest{}, nil
}
