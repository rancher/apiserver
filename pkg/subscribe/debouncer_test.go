package subscribe

import (
	"context"
	"testing"
	"time"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDebouncer(t *testing.T) {
	ctx := context.Background()

	debounceRate := 100 * time.Millisecond

	// 5 events, but we'll expect only two
	count := 5
	eventsCh := make(chan types.APIEvent, count)
	for range count {
		eventsCh <- types.APIEvent{
			Revision: "111",
		}
	}
	time.AfterFunc(200*time.Millisecond, func() {
		close(eventsCh)
	})

	deb := newDebouncer(debounceRate, eventsCh)
	go deb.Run(ctx)

	expectedEvents := []types.APIEvent{
		{
			Name:     string(SubscriptionModeNotification),
			Revision: "111",
		},
		{
			Name:     string(SubscriptionModeNotification),
			Revision: "111",
		},
	}

	var gotEvents []types.APIEvent
	for ev := range deb.NotificationsChan() {
		gotEvents = append(gotEvents, ev)
	}
	assert.Equal(t, expectedEvents, gotEvents)
}
