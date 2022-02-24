package infector

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
)

const (
	_headerKeyDeadline = "deadline-ms"
	_headerKeyTimeout  = "timeout-ms"
	_headerKeyRetry    = "retry-flag"

	RetryUnknown string = "unknown"
	RetryOn      string = "on"
	RetryOff     string = "off"
)

var (
	prefixKey         = "infector-"
	headerKeyDeadline = "deadline-ms"
	headerKeyTimeout  = "timeout-ms"
	headerKeyRetry    = "retry-flag"

	defaultLogger Logger = new(NullLogger)

	ErrInvalidTimeout  = errors.New("invalid timeout")
	ErrInvalidDeadline = errors.New("invalid deadline")
	ErrInvalidRetry    = errors.New("invalid retry")
)

func init() {
	SetPrefixKey(prefixKey)
}

func SetPrefixKey(pre string) {
	if !strings.HasSuffix(pre, "-") {
		pre = pre + "-"
	}
	headerKeyDeadline = pre + _headerKeyDeadline
	headerKeyRetry = pre + _headerKeyRetry
	headerKeyTimeout = pre + _headerKeyTimeout
}

func verifyRetryFlag(r string) bool {
	switch r {
	case RetryUnknown, RetryOn, RetryOff:
		return true
	default:
		return false
	}
}

func InjectHeaderCtx(ctx context.Context, _header interface{}, retry string) {
	header := WrapMapper(_header)

	// set retry
	if verifyRetryFlag(retry) {
		header.Set(headerKeyRetry, retry)
	} else {
		header.Set(headerKeyRetry, RetryUnknown)
	}

	// set deadline
	due, ok := ctx.Deadline()
	if !ok {
		return
	}
	if due.IsZero() {
		return
	}

	header.Set(headerKeyDeadline, formatUnixTime(due))
	header.Set(headerKeyTimeout, "0") // default zero

	diff := due.Sub(time.Now()).Milliseconds()
	if diff > 0 {
		header.Set(headerKeyTimeout, strconv.FormatInt(diff, 10))
	}
}

// Entry
type Entry struct {
	Timeout  time.Duration
	Deadline time.Time
	Retry    string
}

// parseDeadline
func parseDeadline(val string) (time.Time, error) {
	if val == "" {
		return time.Time{}, ErrInvalidDeadline
	}

	ts, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}, ErrInvalidDeadline
	}

	return convTime(ts), nil
}

// parseTimeout
func parseTimeout(val string) (time.Duration, error) {
	if val == "" {
		return 0, ErrInvalidTimeout
	}

	timeout, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, ErrInvalidTimeout
	}

	return convDuration(timeout), nil
}

// parseRetryFlag
func parseRetryFlag(val string) (string, error) {
	retry := RetryUnknown
	if verifyRetryFlag(val) {
		retry = val
	}
	return retry, nil
}

// ParseHeader
func ParseHeader(hdr interface{}) (time.Duration, string, error) {
	header := WrapMapper(hdr)

	retry, err := parseRetryFlag(header.Get(headerKeyRetry))
	if err != nil {
		defaultLogger.Error("parse retry failed, err: " + err.Error())
	}

	timeout, err := parseTimeout(header.Get(headerKeyTimeout))
	if err != nil {
		defaultLogger.Error("parse timeout failed, err: " + err.Error())
	}

	return timeout, retry, err
}

// ParseEntry
func ParseEntry(ctx context.Context, header interface{}) (*Entry, error) {
	timeout, retry, err := ParseHeader(header)
	if err != nil {
		return nil, err
	}

	return &Entry{
		Timeout:  timeout,
		Deadline: time.Now().Add(timeout),
		Retry:    retry,
	}, nil
}

const headerCtx = "infector_header"

// ParseSpanFromHeader header type is in the range of http.header, grpc.metadata and map.
func ParseSpanFromHeader(ctx context.Context, header interface{}) (*SpanContext, error) {
	timeout, retry, err := ParseHeader(header)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, headerCtx, header)
	return NewSpanContext(ctx, timeout, retry), nil
}

// ParseSpanFromCtx
func ParseSpanFromCtx(ctx context.Context) (*SpanContext, error) {
	_value := ctx.Value(headerCtx)
	if _value == nil {
		return nil, errors.New("unset header and retry in header")
	}

	var (
		cancel   = func() {}
		deadline = time.Time{}
	)

	timeout, retry, _ := ParseHeader(_value)
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	return &SpanContext{
		ctx:       ctx,
		cancel:    cancel,
		Deadline:  deadline,
		Timeout:   timeout,
		RetryFlag: retry,
	}, nil
}

// SpanContext
type SpanContext struct {
	ctx    context.Context
	cancel context.CancelFunc

	TimeExists bool
	Deadline   time.Time
	Timeout    time.Duration

	RetryFlag string
}

// NewSpanContext
func NewSpanContext(ctx context.Context, timeout time.Duration, retry string) *SpanContext {
	var (
		cctx     = ctx
		cancel   = func() {}
		deadline = time.Time{}
	)

	if timeout > 0 {
		cctx, cancel = context.WithTimeout(ctx, timeout)
		deadline = time.Now().Add(timeout)
	}

	return &SpanContext{
		ctx:       cctx,
		cancel:    cancel,
		Deadline:  deadline,
		Timeout:   timeout,
		RetryFlag: retry,
	}
}

// GetContext
func (sc *SpanContext) GetContext() context.Context {
	return sc.ctx
}

// GetCancel
func (sc *SpanContext) GetCancel() context.CancelFunc {
	return sc.cancel
}

// GetContextCancel
func (sc *SpanContext) GetContextCancel() (context.Context, context.CancelFunc) {
	return sc.ctx, sc.cancel
}

// cancel
func (sc *SpanContext) Cancel() {
	sc.cancel()
}

// IsRetryON
func (sc *SpanContext) IsRetryON() bool {
	return sc.RetryFlag == RetryOn
}

// IsRetryOff
func (sc *SpanContext) IsRetryOff() bool {
	return sc.RetryFlag == RetryOff
}

// IsRetryUnknown
func (sc *SpanContext) IsRetryUnknown() bool {
	return sc.RetryFlag == RetryUnknown
}

// ContinueRetry
func (sc *SpanContext) ContinueRetry() bool {
	if sc.RetryFlag == RetryOn {
		return true
	}
	if sc.RetryFlag == RetryUnknown { // if retry is unset, allow to continue retry.
		return true
	}

	return false
}

// InjectHeader
func (sc *SpanContext) InjectHeader(mapper interface{}) {
	InjectHeaderCtx(sc.ctx, mapper, sc.RetryFlag)
}

// GetHttpHeader
func (sc *SpanContext) GetHttpHeader(hdrs ...http.Header) http.Header {
	header := http.Header{}
	InjectHeaderCtx(sc.ctx, header, sc.RetryFlag)

	if len(hdrs) == 0 {
		return header
	}
	for k, vs := range hdrs[0] {
		for _, v := range vs {
			header.Set(k, v)
		}
	}
	return header
}

// GetGrpcMetadata
func (sc *SpanContext) GetGrpcMetadata(mds ...metadata.MD) metadata.MD {
	md := metadata.Pairs()
	InjectHeaderCtx(sc.ctx, md, sc.RetryFlag)

	if len(mds) == 0 {
		return md
	}
	for k, vs := range mds[0] {
		md.Set(k, vs...)
	}
	return md
}

// NotTimeout
func (sc *SpanContext) NotTimeout() bool {
	return !sc.ReachTimeout()
}

// ReachTimeout
func (sc *SpanContext) ReachTimeout() bool {
	if sc.Deadline.IsZero() {
		return true
	}
	if time.Now().After(sc.Deadline) {
		return true
	}
	return false
}

// convUnixTime time.time to mills int64
func convUnixTime(ts time.Time) int64 {
	return ts.UnixNano() / int64(time.Millisecond)
}

// convTime mills int convert to time.Time, code from go1.17 time.UnixMilli().
func convTime(msec int64) time.Time {
	return time.Unix(msec/1e3, (msec%1e3)*1e6)
}

// convDuration
func convDuration(msec int64) time.Duration {
	return time.Duration(msec) * time.Millisecond
}

// formatRetryFlag
func formatRetryFlag(b bool) string {
	if b {
		return RetryOn
	}
	return RetryOff
}

// formatUnixTime
func formatUnixTime(ts time.Time) string {
	return strconv.FormatInt(convUnixTime(ts), 10)
}

// logger
type Logger interface {
	Error(string)
}

type NullLogger struct{}

func (l *NullLogger) Error(msg string) {}

func SetLogger(logger Logger) {
	defaultLogger = logger
}
