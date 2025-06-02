package middleware

import "net/http"

// SetCORSHeaders is a middleware function that offers CORS headers for an
// OPTION request. This should be used earliest in the base router.
func SetCORSHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodOptions:
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			writer.Header().Set("Access-Control-Allow-Methods", "POST,GET,PATCH")
			writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
			writer.Header().Set("Access-Control-Max-Age", "3600")
			writer.WriteHeader(http.StatusNoContent)
		case http.MethodHead:
			writer.WriteHeader(http.StatusNoContent)
		default:
			writer.Header().Set("Access-Control-Allow-Origin", "*")
			next.ServeHTTP(writer, request)
		}
	})
}
