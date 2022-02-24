package infector

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	ErrHeaderRequestTimeout = errors.New("the timeout-ms value in header is already timeout.")
	ErrRequestTimeout       = errors.New("request timeout.")
)

type HttpOption struct {
	header   http.Header
	response interface{}
}

var defaultHttpOption = HttpOption{
	header: http.Header{},
	response: map[string]string{
		"error": ErrHeaderRequestTimeout.Error(),
	},
}

type OptionGinFunc func(op *HttpOption)

func WithGinResponse(header http.Header, obj interface{}) OptionGinFunc {
	return func(op *HttpOption) {
		op.header = header
		op.response = obj
	}
}

// GinMiddleware gin middleware
func GinMiddleware(opts ...OptionGinFunc) gin.HandlerFunc {
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

		if span.ReachTimeout() {
			for k, v := range option.header {
				c.Writer.Header().Set(k, v[0])
			}
			c.JSON(200, option.response)
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

// ServerInterceptor grpc server wrapper
func ServerInterceptor() grpc.UnaryServerInterceptor {
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
		if span.ReachTimeout() {
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
