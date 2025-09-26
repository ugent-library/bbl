package httperr

import (
	"fmt"
	"net/http"
)

var (
	// 4xx
	BadRequest                   = &StatusError{StatusCode: http.StatusBadRequest}
	Unauthorized                 = &StatusError{StatusCode: http.StatusUnauthorized}
	PaymentRequired              = &StatusError{StatusCode: http.StatusPaymentRequired}
	Forbidden                    = &StatusError{StatusCode: http.StatusForbidden}
	NotFound                     = &StatusError{StatusCode: http.StatusNotFound}
	MethodNotAllowed             = &StatusError{StatusCode: http.StatusMethodNotAllowed}
	NotAcceptable                = &StatusError{StatusCode: http.StatusNotAcceptable}
	ProxyAuthRequired            = &StatusError{StatusCode: http.StatusProxyAuthRequired}
	RequestTimeout               = &StatusError{StatusCode: http.StatusRequestTimeout}
	Conflict                     = &StatusError{StatusCode: http.StatusConflict}
	Gone                         = &StatusError{StatusCode: http.StatusGone}
	LengthRequired               = &StatusError{StatusCode: http.StatusLengthRequired}
	PreconditionFailed           = &StatusError{StatusCode: http.StatusPreconditionFailed}
	RequestEntityTooLarge        = &StatusError{StatusCode: http.StatusRequestEntityTooLarge}
	RequestURITooLong            = &StatusError{StatusCode: http.StatusRequestURITooLong}
	UnsupportedMediaType         = &StatusError{StatusCode: http.StatusUnsupportedMediaType}
	RequestedRangeNotSatisfiable = &StatusError{StatusCode: http.StatusRequestedRangeNotSatisfiable}
	ExpectationFailed            = &StatusError{StatusCode: http.StatusExpectationFailed}
	Teapot                       = &StatusError{StatusCode: http.StatusTeapot}
	MisdirectedRequest           = &StatusError{StatusCode: http.StatusMisdirectedRequest}
	UnprocessableEntity          = &StatusError{StatusCode: http.StatusUnprocessableEntity}
	Locked                       = &StatusError{StatusCode: http.StatusLocked}
	FailedDependency             = &StatusError{StatusCode: http.StatusFailedDependency}
	TooEarly                     = &StatusError{StatusCode: http.StatusTooEarly}
	UpgradeRequired              = &StatusError{StatusCode: http.StatusUpgradeRequired}
	PreconditionRequired         = &StatusError{StatusCode: http.StatusPreconditionRequired}
	TooManyRequests              = &StatusError{StatusCode: http.StatusTooManyRequests}
	RequestHeaderFieldsTooLarge  = &StatusError{StatusCode: http.StatusRequestHeaderFieldsTooLarge}
	UnavailableForLegalReasons   = &StatusError{StatusCode: http.StatusUnavailableForLegalReasons}
	// 5xx
	InternalServerError           = &StatusError{StatusCode: http.StatusInternalServerError}
	NotImplemented                = &StatusError{StatusCode: http.StatusNotImplemented}
	BadGateway                    = &StatusError{StatusCode: http.StatusBadGateway}
	ServiceUnavailable            = &StatusError{StatusCode: http.StatusServiceUnavailable}
	GatewayTimeout                = &StatusError{StatusCode: http.StatusGatewayTimeout}
	HTTPVersionNotSupported       = &StatusError{StatusCode: http.StatusHTTPVersionNotSupported}
	VariantAlsoNegotiates         = &StatusError{StatusCode: http.StatusVariantAlsoNegotiates}
	InsufficientStorage           = &StatusError{StatusCode: http.StatusInsufficientStorage}
	LoopDetected                  = &StatusError{StatusCode: http.StatusLoopDetected}
	NotExtended                   = &StatusError{StatusCode: http.StatusNotExtended}
	NetworkAuthenticationRequired = &StatusError{StatusCode: http.StatusNetworkAuthenticationRequired}
)

type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("http error %d: %s", e.StatusCode, http.StatusText(e.StatusCode))
}
