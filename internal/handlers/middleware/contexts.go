package middleware

import (
	"context"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

type contextKey int

const (
	ctxMailItemKey contextKey = iota
	ctxMailItemAttachmentKey
	ctxUserKey
)

// AttachMailItem ...
func AttachMailItem(ctx context.Context, item model.MailItem) context.Context {
	return context.WithValue(ctx, ctxMailItemKey, item)
}

// GetMailItem ...
func GetMailItem(ctx context.Context) *model.MailItem {
	val := ctx.Value(ctxMailItemKey)
	if val == nil {
		return nil
	}

	item, ok := val.(model.MailItem)
	if !ok {
		return nil
	}

	return &item
}

// AttachMailAttachment ...
func AttachMailAttachment(ctx context.Context, attachment model.Attachment) context.Context {
	return context.WithValue(ctx, ctxMailItemAttachmentKey, attachment)
}

// GetMailItemAttachment ...
func GetMailItemAttachment(ctx context.Context) *model.Attachment {
	val := ctx.Value(ctxMailItemAttachmentKey)
	if val == nil {
		return nil
	}

	item, ok := val.(model.Attachment)
	if !ok {
		return nil
	}

	return &item
}

func AttachUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, ctxUserKey, user)
}

func GetUser(ctx context.Context) *string {
	val := ctx.Value(ctxUserKey)
	if val == nil {
		return nil
	}

	user, ok := val.(string)
	if !ok {
		return nil
	}

	return &user
}
