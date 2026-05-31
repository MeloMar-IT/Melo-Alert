package domain

import (
	"context"
)

type Repository interface {
	StoreRawEvent(ctx context.Context, event *RawEvent) error
	UpsertAlert(ctx context.Context, alert *Alert) error
	AppendAlertEvent(ctx context.Context, event *AlertEvent) error
}
