package endpoint

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/lixichongAAA/seckill/oauth-service/model"
	"github.com/lixichongAAA/seckill/oauth-service/service"
)

// CalculateEndpoint define endpoint
type OAuth2Endpoints struct {
	TokenEndpoint          endpoint.Endpoint
	CheckTokenEndpoint     endpoint.Endpoint
	GRPCCheckTokenEndpoint endpoint.Endpoint
	HealthCheckEndpoint    endpoint.Endpoint
}

// 在请求正式进入到 Endpoint 之前，我们还要验证 context 的客户端信息是否存在
// 客户端验证中间件如下
func MakeClientAuthorizationMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {

		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			// 请求上下文是否存在错误
			if err, ok := ctx.Value(OAuth2ErrorKey).(error); ok {
				return nil, err
			}
			// 验证客户端信息是否存在，不存在则返回异常
			if _, ok := ctx.Value(OAuth2ClientDetailsKey).(*model.ClientDetails); !ok {
				return nil, ErrInvalidClientRequest
			}
			return next(ctx, request)
		}
	}
}

// 在进入到 Endpoint 之前统一验证 context 中的 OAuth2Details 是否存在
// 如果不存在,说明请求没有经过认证,拒绝请求访问.
func MakeOAuth2AuthorizationMiddleware(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {

		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			if err, ok := ctx.Value(OAuth2ErrorKey).(error); ok {
				return nil, err
			}
			// 检查请求上下文中是否存在 OAuth2Details(即用户和客户端信息)
			if _, ok := ctx.Value(OAuth2DetailsKey).(*model.OAuth2Details); !ok {
				return nil, ErrInvalidUserRequest
			}
			return next(ctx, request)
		}
	}
}

// 鉴权
func MakeAuthorityAuthorizationdMiddleware(authority string, logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {

		return func(ctx context.Context, request interface{}) (response interface{}, err error) {

			if err, ok := ctx.Value(OAuth2ErrorKey).(error); ok {
				return nil, err
			}
			// 获取 Context 中的用户和客户端信息
			if details, ok := ctx.Value(OAuth2DetailsKey).(*model.OAuth2Details); !ok {
				return nil, ErrInvalidClientRequest
			} else {
				// 权限检查
				for _, value := range details.User.Authorities {
					if value == authority {
						return next(ctx, request)
					}
				}
				return nil, ErrNotPermit
			}
		}
	}
}

const (
	OAuth2DetailsKey       = "OAuth2Details"
	OAuth2ClientDetailsKey = "OAuth2ClientDetails"
	OAuth2ErrorKey         = "OAuth2Error"
)

var (
	ErrInvalidClientRequest = errors.New("invalid client message")
	ErrInvalidUserRequest   = errors.New("invalid user message")
	ErrNotPermit            = errors.New("not permit")
)

type TokenRequest struct {
	GrantType string
	Reader    *http.Request
}

type TokenResponse struct {
	AccessToken *model.OAuth2Token `json:"access_token"`
	Error       string             `json:"error"`
}

// make endpoint
// MakeTokenEndpoint 端点从 context 中获取到请求客户端的信息，然后委托 TokenGrant 根据授权类型和用户凭证
// 为客户端生成访问令牌，然后返回
func MakeTokenEndpoint(svc service.TokenGranter, clientService service.ClientDetailsService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*TokenRequest)
		token, err := svc.Grant(ctx, req.GrantType, ctx.Value(OAuth2ClientDetailsKey).(*model.ClientDetails), req.Reader)
		var errString = ""
		if err != nil {
			errString = err.Error()
		}

		return TokenResponse{
			AccessToken: token,
			Error:       errString,
		}, nil
	}
}

type CheckTokenRequest struct {
	Token         string
	ClientDetails model.ClientDetails
}

type CheckTokenResponse struct {
	OAuthDetails *model.OAuth2Details `json:"o_auth_details"`
	Error        string               `json:"error"`
}

// 通过将请求中的 tokenvalue 传递给 ToeknService.GetOAuth2DetailsByAccessToken 方法
// 以验证 token 的有效性
func MakeCheckTokenEndpoint(svc service.TokenService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(*CheckTokenRequest)
		tokenDetails, err := svc.GetOAuth2DetailsByAccessToken(req.Token)

		var errString = ""
		if err != nil {
			errString = err.Error()
		}

		return CheckTokenResponse{
			OAuthDetails: tokenDetails,
			Error:        errString,
		}, nil
	}
}

type SimpleRequest struct {
}

type SimpleResponse struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

type AdminRequest struct {
}

type AdminResponse struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}

// HealthRequest 健康检查请求结构
type HealthRequest struct{}

// HealthResponse 健康检查响应结构
type HealthResponse struct {
	Status bool `json:"status"`
}

// MakeHealthCheckEndpoint 创建健康检查Endpoint
func MakeHealthCheckEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		status := svc.HealthCheck()
		return HealthResponse{
			Status: status,
		}, nil
	}
}
