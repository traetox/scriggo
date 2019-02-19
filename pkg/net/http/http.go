// Go version: go1.11.5

package http

import "scrigo"
import "reflect"
import original "net/http"

var Package = scrigo.Package{
	"CanonicalHeaderKey": original.CanonicalHeaderKey,
	"Client": reflect.TypeOf(original.Client{}),
	"CloseNotifier": reflect.TypeOf((*original.CloseNotifier)(nil)).Elem(),
	"ConnState": reflect.TypeOf(original.ConnState(int(0))),
	"Cookie": reflect.TypeOf(original.Cookie{}),
	"CookieJar": reflect.TypeOf((*original.CookieJar)(nil)).Elem(),
	"DefaultClient": &original.DefaultClient,
	"DefaultMaxHeaderBytes": scrigo.Constant(original.DefaultMaxHeaderBytes, nil),
	"DefaultMaxIdleConnsPerHost": scrigo.Constant(original.DefaultMaxIdleConnsPerHost, nil),
	"DefaultServeMux": &original.DefaultServeMux,
	"DefaultTransport": &original.DefaultTransport,
	"DetectContentType": original.DetectContentType,
	"Dir": reflect.TypeOf(""),
	"ErrAbortHandler": &original.ErrAbortHandler,
	"ErrBodyNotAllowed": &original.ErrBodyNotAllowed,
	"ErrBodyReadAfterClose": &original.ErrBodyReadAfterClose,
	"ErrContentLength": &original.ErrContentLength,
	"ErrHandlerTimeout": &original.ErrHandlerTimeout,
	"ErrHeaderTooLong": &original.ErrHeaderTooLong,
	"ErrHijacked": &original.ErrHijacked,
	"ErrLineTooLong": &original.ErrLineTooLong,
	"ErrMissingBoundary": &original.ErrMissingBoundary,
	"ErrMissingContentLength": &original.ErrMissingContentLength,
	"ErrMissingFile": &original.ErrMissingFile,
	"ErrNoCookie": &original.ErrNoCookie,
	"ErrNoLocation": &original.ErrNoLocation,
	"ErrNotMultipart": &original.ErrNotMultipart,
	"ErrNotSupported": &original.ErrNotSupported,
	"ErrServerClosed": &original.ErrServerClosed,
	"ErrShortBody": &original.ErrShortBody,
	"ErrSkipAltProtocol": &original.ErrSkipAltProtocol,
	"ErrUnexpectedTrailer": &original.ErrUnexpectedTrailer,
	"ErrUseLastResponse": &original.ErrUseLastResponse,
	"ErrWriteAfterFlush": &original.ErrWriteAfterFlush,
	"Error": original.Error,
	"File": reflect.TypeOf((*original.File)(nil)).Elem(),
	"FileServer": original.FileServer,
	"FileSystem": reflect.TypeOf((*original.FileSystem)(nil)).Elem(),
	"Flusher": reflect.TypeOf((*original.Flusher)(nil)).Elem(),
	"Get": original.Get,
	"Handle": original.Handle,
	"HandleFunc": original.HandleFunc,
	"Handler": reflect.TypeOf((*original.Handler)(nil)).Elem(),
	"HandlerFunc": reflect.TypeOf((original.HandlerFunc)(nil)),
	"Head": original.Head,
	"Header": reflect.TypeOf((original.Header)(nil)),
	"Hijacker": reflect.TypeOf((*original.Hijacker)(nil)).Elem(),
	"ListenAndServe": original.ListenAndServe,
	"ListenAndServeTLS": original.ListenAndServeTLS,
	"LocalAddrContextKey": &original.LocalAddrContextKey,
	"MaxBytesReader": original.MaxBytesReader,
	"MethodConnect": scrigo.Constant(original.MethodConnect, nil),
	"MethodDelete": scrigo.Constant(original.MethodDelete, nil),
	"MethodGet": scrigo.Constant(original.MethodGet, nil),
	"MethodHead": scrigo.Constant(original.MethodHead, nil),
	"MethodOptions": scrigo.Constant(original.MethodOptions, nil),
	"MethodPatch": scrigo.Constant(original.MethodPatch, nil),
	"MethodPost": scrigo.Constant(original.MethodPost, nil),
	"MethodPut": scrigo.Constant(original.MethodPut, nil),
	"MethodTrace": scrigo.Constant(original.MethodTrace, nil),
	"NewFileTransport": original.NewFileTransport,
	"NewRequest": original.NewRequest,
	"NewServeMux": original.NewServeMux,
	"NoBody": &original.NoBody,
	"NotFound": original.NotFound,
	"NotFoundHandler": original.NotFoundHandler,
	"ParseHTTPVersion": original.ParseHTTPVersion,
	"ParseTime": original.ParseTime,
	"Post": original.Post,
	"PostForm": original.PostForm,
	"ProtocolError": reflect.TypeOf(original.ProtocolError{}),
	"ProxyFromEnvironment": original.ProxyFromEnvironment,
	"ProxyURL": original.ProxyURL,
	"PushOptions": reflect.TypeOf(original.PushOptions{}),
	"Pusher": reflect.TypeOf((*original.Pusher)(nil)).Elem(),
	"ReadRequest": original.ReadRequest,
	"ReadResponse": original.ReadResponse,
	"Redirect": original.Redirect,
	"RedirectHandler": original.RedirectHandler,
	"Request": reflect.TypeOf(original.Request{}),
	"Response": reflect.TypeOf(original.Response{}),
	"ResponseWriter": reflect.TypeOf((*original.ResponseWriter)(nil)).Elem(),
	"RoundTripper": reflect.TypeOf((*original.RoundTripper)(nil)).Elem(),
	"SameSite": reflect.TypeOf(original.SameSite(int(0))),
	"SameSiteDefaultMode": scrigo.Constant(original.SameSiteDefaultMode, nil),
	"SameSiteLaxMode": scrigo.Constant(original.SameSiteLaxMode, nil),
	"SameSiteStrictMode": scrigo.Constant(original.SameSiteStrictMode, nil),
	"Serve": original.Serve,
	"ServeContent": original.ServeContent,
	"ServeFile": original.ServeFile,
	"ServeMux": reflect.TypeOf(original.ServeMux{}),
	"ServeTLS": original.ServeTLS,
	"Server": reflect.TypeOf(original.Server{}),
	"ServerContextKey": &original.ServerContextKey,
	"SetCookie": original.SetCookie,
	"StateActive": scrigo.Constant(original.StateActive, nil),
	"StateClosed": scrigo.Constant(original.StateClosed, nil),
	"StateHijacked": scrigo.Constant(original.StateHijacked, nil),
	"StateIdle": scrigo.Constant(original.StateIdle, nil),
	"StateNew": scrigo.Constant(original.StateNew, nil),
	"StatusAccepted": scrigo.Constant(original.StatusAccepted, nil),
	"StatusAlreadyReported": scrigo.Constant(original.StatusAlreadyReported, nil),
	"StatusBadGateway": scrigo.Constant(original.StatusBadGateway, nil),
	"StatusBadRequest": scrigo.Constant(original.StatusBadRequest, nil),
	"StatusConflict": scrigo.Constant(original.StatusConflict, nil),
	"StatusContinue": scrigo.Constant(original.StatusContinue, nil),
	"StatusCreated": scrigo.Constant(original.StatusCreated, nil),
	"StatusExpectationFailed": scrigo.Constant(original.StatusExpectationFailed, nil),
	"StatusFailedDependency": scrigo.Constant(original.StatusFailedDependency, nil),
	"StatusForbidden": scrigo.Constant(original.StatusForbidden, nil),
	"StatusFound": scrigo.Constant(original.StatusFound, nil),
	"StatusGatewayTimeout": scrigo.Constant(original.StatusGatewayTimeout, nil),
	"StatusGone": scrigo.Constant(original.StatusGone, nil),
	"StatusHTTPVersionNotSupported": scrigo.Constant(original.StatusHTTPVersionNotSupported, nil),
	"StatusIMUsed": scrigo.Constant(original.StatusIMUsed, nil),
	"StatusInsufficientStorage": scrigo.Constant(original.StatusInsufficientStorage, nil),
	"StatusInternalServerError": scrigo.Constant(original.StatusInternalServerError, nil),
	"StatusLengthRequired": scrigo.Constant(original.StatusLengthRequired, nil),
	"StatusLocked": scrigo.Constant(original.StatusLocked, nil),
	"StatusLoopDetected": scrigo.Constant(original.StatusLoopDetected, nil),
	"StatusMethodNotAllowed": scrigo.Constant(original.StatusMethodNotAllowed, nil),
	"StatusMisdirectedRequest": scrigo.Constant(original.StatusMisdirectedRequest, nil),
	"StatusMovedPermanently": scrigo.Constant(original.StatusMovedPermanently, nil),
	"StatusMultiStatus": scrigo.Constant(original.StatusMultiStatus, nil),
	"StatusMultipleChoices": scrigo.Constant(original.StatusMultipleChoices, nil),
	"StatusNetworkAuthenticationRequired": scrigo.Constant(original.StatusNetworkAuthenticationRequired, nil),
	"StatusNoContent": scrigo.Constant(original.StatusNoContent, nil),
	"StatusNonAuthoritativeInfo": scrigo.Constant(original.StatusNonAuthoritativeInfo, nil),
	"StatusNotAcceptable": scrigo.Constant(original.StatusNotAcceptable, nil),
	"StatusNotExtended": scrigo.Constant(original.StatusNotExtended, nil),
	"StatusNotFound": scrigo.Constant(original.StatusNotFound, nil),
	"StatusNotImplemented": scrigo.Constant(original.StatusNotImplemented, nil),
	"StatusNotModified": scrigo.Constant(original.StatusNotModified, nil),
	"StatusOK": scrigo.Constant(original.StatusOK, nil),
	"StatusPartialContent": scrigo.Constant(original.StatusPartialContent, nil),
	"StatusPaymentRequired": scrigo.Constant(original.StatusPaymentRequired, nil),
	"StatusPermanentRedirect": scrigo.Constant(original.StatusPermanentRedirect, nil),
	"StatusPreconditionFailed": scrigo.Constant(original.StatusPreconditionFailed, nil),
	"StatusPreconditionRequired": scrigo.Constant(original.StatusPreconditionRequired, nil),
	"StatusProcessing": scrigo.Constant(original.StatusProcessing, nil),
	"StatusProxyAuthRequired": scrigo.Constant(original.StatusProxyAuthRequired, nil),
	"StatusRequestEntityTooLarge": scrigo.Constant(original.StatusRequestEntityTooLarge, nil),
	"StatusRequestHeaderFieldsTooLarge": scrigo.Constant(original.StatusRequestHeaderFieldsTooLarge, nil),
	"StatusRequestTimeout": scrigo.Constant(original.StatusRequestTimeout, nil),
	"StatusRequestURITooLong": scrigo.Constant(original.StatusRequestURITooLong, nil),
	"StatusRequestedRangeNotSatisfiable": scrigo.Constant(original.StatusRequestedRangeNotSatisfiable, nil),
	"StatusResetContent": scrigo.Constant(original.StatusResetContent, nil),
	"StatusSeeOther": scrigo.Constant(original.StatusSeeOther, nil),
	"StatusServiceUnavailable": scrigo.Constant(original.StatusServiceUnavailable, nil),
	"StatusSwitchingProtocols": scrigo.Constant(original.StatusSwitchingProtocols, nil),
	"StatusTeapot": scrigo.Constant(original.StatusTeapot, nil),
	"StatusTemporaryRedirect": scrigo.Constant(original.StatusTemporaryRedirect, nil),
	"StatusText": original.StatusText,
	"StatusTooManyRequests": scrigo.Constant(original.StatusTooManyRequests, nil),
	"StatusUnauthorized": scrigo.Constant(original.StatusUnauthorized, nil),
	"StatusUnavailableForLegalReasons": scrigo.Constant(original.StatusUnavailableForLegalReasons, nil),
	"StatusUnprocessableEntity": scrigo.Constant(original.StatusUnprocessableEntity, nil),
	"StatusUnsupportedMediaType": scrigo.Constant(original.StatusUnsupportedMediaType, nil),
	"StatusUpgradeRequired": scrigo.Constant(original.StatusUpgradeRequired, nil),
	"StatusUseProxy": scrigo.Constant(original.StatusUseProxy, nil),
	"StatusVariantAlsoNegotiates": scrigo.Constant(original.StatusVariantAlsoNegotiates, nil),
	"StripPrefix": original.StripPrefix,
	"TimeFormat": scrigo.Constant(original.TimeFormat, nil),
	"TimeoutHandler": original.TimeoutHandler,
	"TrailerPrefix": scrigo.Constant(original.TrailerPrefix, nil),
	"Transport": reflect.TypeOf(original.Transport{}),
}
