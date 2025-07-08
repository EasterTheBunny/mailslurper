package app_test

import (
	"log/slog"
	"net/http"
	"net/smtp"
	"testing"
	"time"

	"github.com/adampresley/webframework/sanitizer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mailslurper/mailslurper/v2/internal/app"
	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/mocks"
	"github.com/mailslurper/mailslurper/v2/internal/model"
	appsmtp "github.com/mailslurper/mailslurper/v2/internal/smtp"
)

func TestSMTPService_Lifecycle(t *testing.T) {
	t.Parallel()

	config := &io.Config{
		SMTP: io.ListenConfig{
			Address: "127.0.0.1",
			Port:    0,
		},
	}

	xss := sanitizer.NewXSSService()
	db := new(mocks.MockMailWriter)
	logger := slog.New(slog.NewTextHandler(tWriter{t: t}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	svc := app.NewSMTPService(config, xss, db, logger)

	go func() {
		timer := time.NewTimer(time.Second)

		// wait for 1 second or until test context is done, whichever comes first
		select {
		case <-timer.C:
		case <-t.Context().Done():
		}

		require.NoError(t, svc.Close())
	}()

	assert.ErrorIs(t, svc.Start(), appsmtp.ErrServerClosed)
}

func TestSMTPService_SendMail(t *testing.T) {
	t.Parallel()

	config := &io.Config{
		MaxWorkers: 5,
		SMTP: io.ListenConfig{
			Address: "127.0.0.1",
			Port:    0, // randomly selects port
		},
	}

	xss := sanitizer.NewXSSService()
	db := new(mocks.MockMailWriter)
	logger := slog.New(slog.NewTextHandler(tWriter{t: t}, &slog.HandlerOptions{Level: slog.LevelError}))

	svc := app.NewSMTPService(config, xss, db, logger)

	t.Cleanup(func() {
		assert.NoError(t, svc.Close())
	})

	go func() {
		assert.ErrorIs(t, svc.Start(), appsmtp.ErrServerClosed)
	}()

	chSave := make(chan struct{}, 1)

	from := "one@example.com"
	to1 := "recipient@example.net"
	to2 := "three@example.com"
	msg := []byte("To: recipient@example.net\r\n" +
		"Subject: discount Gophers!\r\n" +
		"Date: 02 Jan 2006 15:04:05 -0700\r\n" +
		"\r\n" +
		"This is the email body.\r\n")

	db.EXPECT().StoreMail(mock.MatchedBy(func(item *model.MailItem) bool {
		require.NotNil(t, item)

		assert.Equal(t, from, item.FromAddress)
		assert.Contains(t, item.ToAddresses, to1)
		assert.Contains(t, item.ToAddresses, to2)

		chSave <- struct{}{}

		return true
	})).Return(nil)

	time.Sleep(time.Second)
	require.NoError(t, smtp.SendMail(svc.Addr().String(), nil, from, []string{to1, to2}, msg))
	time.Sleep(time.Second)

	select {
	case <-chSave:
	case <-t.Context().Done():
		t.Fail()
	}

	db.AssertExpectations(t)
}

func TestHTTPService_Lifecycle(t *testing.T) {
	t.Parallel()

	config := &io.Config{
		Public: io.ListenConfig{
			Address: "127.0.0.1",
			Port:    0,
		},
	}

	db := new(mocks.MockPersistance)
	logger := slog.New(slog.NewTextHandler(tWriter{t: t}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	svc := app.NewHTTPService(&app.HTTPServiceConfig{
		Version: "test",
		Data:    db,
		Config:  config,
		Logger:  logger,
	})

	go func() {
		timer := time.NewTimer(time.Second)

		// wait for 1 second or until test context is done, whichever comes first
		select {
		case <-timer.C:
		case <-t.Context().Done():
		}

		require.NoError(t, svc.Close())
	}()

	assert.ErrorIs(t, svc.Start(), http.ErrServerClosed)
}

type tWriter struct {
	t *testing.T
}

func (w tWriter) Write(bts []byte) (int, error) {
	w.t.Log(string(bts))

	return len(bts), nil
}
