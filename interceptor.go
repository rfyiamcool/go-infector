package infector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	ErrHeaderRequestTimeout = errors.New("the timeout-ms value in header is 0, not enough time.")
	ErrRequestTimeout       = errors.New("request timeout.")
)

type HttpOption struct {
	header     http.Header
	response   interface{}
	LeastQuota time.Duration
}

var defaultHttpOption = HttpOption{
	header: http.Header{},
	response: map[string]string{
		"error": ErrHeaderRequestTimeout.Error(),
	},
	LeastQuota: 0,
}

// SetDefaultHttpOption the function call before app register middleware.
func SetDefaultHttpOption(ho HttpOption) {
	defaultHttpOption = ho
}

type optionGinFunc func(op *HttpOption)

func WithGinResponse(header http.Header, obj interface{}) optionGinFunc {
	return func(op *HttpOption) {
		op.header = header
		op.response = obj
	}
}

func WithGinLeastQuota(quota time.Duration) optionGinFunc {
	return func(op *HttpOption) {
		op.LeastQuota = quota
	}
}

// GinMiddleware gin middleware
func GinMiddleware(opts ...optionGinFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		option := defaultHttpOption
		for _, opt := range opts {
			opt(&option)
		}

		span, err := ParseSpanFromHeader(c.Request.Context(), c.Request.Header)
		if err != nil {
			c.Next()
			return
		}

		c.Request = c.Request.WithContext(span.ctx)
		defer span.Cancel()

		exception := func() {
			for k, v := range option.header {
				c.Writer.Header().Set(k, v[0])
			}
			c.JSON(200, option.response)
			c.Abort()
		}

		if option.LeastQuota > 0 && !span.PromiseLeastQuota(option.LeastQuota) {
			exception()
			return
		}
		if option.LeastQuota == 0 && span.ReachTimeout() {
			exception()
			return
		}

		c.Next()
	}
}

// HttpMiddleware
func HttpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, err := ParseSpanFromHeader(r.Context(), r.Header)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		r = r.WithContext(span.ctx)
		defer span.Cancel()

		if span.ReachTimeout() {
			bs, _ := json.Marshal(defaultHttpOption.response)
			w.Write(bs)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type GrpcUnaryOption struct {
	header     http.Header
	leastQuota time.Duration
}

var defaultGrpcOption = GrpcUnaryOption{
	header:     http.Header{},
	leastQuota: 0,
}

// SetDefaultGrpcOption
func SetDefaultGrpcOption(dop GrpcUnaryOption) {
	defaultGrpcOption = dop
}

type optionGrpcUnaryFunc func(op *GrpcUnaryOption)

func WithGrpcResponse(header http.Header, obj interface{}) optionGrpcUnaryFunc {
	return func(op *GrpcUnaryOption) {
		op.header = header
	}
}

func WithGrpcLeastQuota(quota time.Duration) optionGrpcUnaryFunc {
	return func(op *GrpcUnaryOption) {
		op.leastQuota = quota
	}
}

// GrpcServerInterceptor grpc server wrapper
func GrpcServerInterceptor(opts ...optionGrpcUnaryFunc) grpc.UnaryServerInterceptor {
	option := defaultGrpcOption
	for _, opt := range opts {
		opt(&option)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		span, err := ParseSpanFromHeader(ctx, md)
		if err != nil {
			return handler(ctx, req)
		}

		defer span.Cancel()

		if option.leastQuota > 0 && !span.PromiseLeastQuota(option.leastQuota) {
			return nil, ErrHeaderRequestTimeout
		}
		if option.leastQuota == 0 && span.ReachTimeout() {
			return nil, ErrHeaderRequestTimeout
		}

		return handler(ctx, req)
	}
}

type RedisHook struct{}

var _ redis.Hook = RedisHook{}

func NewRedisHook() redis.Hook {
	return &RedisHook{}
}

func (hook RedisHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	span, err := ParseSpanFromCtx(ctx)
	if err != nil {
		return ctx, nil
	}

	if span.ReachTimeout() {
		return ctx, ErrRequestTimeout
	}

	return ctx, nil
}

func (hook RedisHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	return nil
}

func (hook RedisHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	span, err := ParseSpanFromCtx(ctx)
	if err != nil {
		return ctx, nil
	}

	if span.ReachTimeout() {
		return ctx, ErrRequestTimeout
	}

	return ctx, nil
}

func (hook RedisHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	return nil
}
