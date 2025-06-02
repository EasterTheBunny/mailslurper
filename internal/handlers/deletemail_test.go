package handlers_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"

	"github.com/mailslurper/mailslurper/v2/internal/handlers"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/requests"
	"github.com/mailslurper/mailslurper/v2/internal/mocks"
)

func TestDeleteMail_InvalidMethod(t *testing.T) {
	t.Parallel()

	logger := slog.NewLogLogger(slog.DiscardHandler, slog.LevelDebug)
	handler := handlers.DeleteMail(nil, logger)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "response code should match expected")
	assert.Contains(t, recorder.Body.String(), `method not allowed`)
	assert.Contains(t, recorder.Body.String(), `"errors"`)
}

func TestDeleteMail_InvalidParam(t *testing.T) {
	t.Parallel()

	logger := slog.NewLogLogger(slog.DiscardHandler, slog.LevelDebug)
	router := chi.NewRouter()
	handler := handlers.DeleteMail(nil, logger)

	router.Delete("/mail", handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/mail?prune=invalid", nil)

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "response code should match expected")
	assert.Contains(t, recorder.Body.String(), `invalid input`)
	assert.Contains(t, recorder.Body.String(), `"errors"`)
}

func TestDeleteMail_Success(t *testing.T) {
	t.Parallel()

	for idx := range requests.PruneOptions {
		code := requests.PruneOptions[idx].PruneCode

		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()

			logger := slog.NewLogLogger(slog.DiscardHandler, slog.LevelDebug)
			mData := new(mocks.MockMailRemover)

			mData.EXPECT().DeleteMailsAfterDate(code.ConvertToDate()).Return(0, nil)

			router := chi.NewRouter()
			handler := handlers.DeleteMail(mData, logger)

			router.Delete("/mail", handler)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/mail?prune=%s", code.String()), nil)

			router.ServeHTTP(recorder, request)

			assert.Equal(t, http.StatusOK, recorder.Code, "response code should match expected")
			assert.Equal(t, recorder.Body.String(), "0")

			mData.AssertExpectations(t)
		})
	}
}
