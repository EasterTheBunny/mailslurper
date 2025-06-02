package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/middleware"
)

func TestSetCORSHeaders(t *testing.T) {
	t.Parallel()

	// create a handler to use as "next" which will verify the request
	nextHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})
	router := chi.NewRouter()

	router.Route("/path", func(router chi.Router) {
		router.Use(middleware.SetCORSHeaders)
		router.Get("/sub-path", nextHandler)
	})

	tests := []struct {
		name   string
		method string
		status int
	}{
		{name: "Options", method: http.MethodOptions, status: http.StatusNoContent},
		{name: "Default", method: http.MethodGet, status: http.StatusOK},
	}

	for idx := range tests {
		test := tests[idx]

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, "http://testing.org/path/sub-path", nil)
			request = request.WithContext(t.Context())

			router.ServeHTTP(recorder, request)
			require.Equal(t, test.status, recorder.Code)

			result := recorder.Result()

			t.Cleanup(func() {
				result.Body.Close()
			})
			assert.Equal(t, "*", result.Header.Get("Access-Control-Allow-Origin"))
		})
	}
}
