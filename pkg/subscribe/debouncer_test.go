package subscribe

import (
	"context"
	"fmt"
	"strconv"
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
	for i := range count {
		eventsCh <- types.APIEvent{
			Revision: strconv.Itoa(i),
		}

	}
	time.AfterFunc(200*time.Millisecond, func() {
		close(eventsCh)
	})

	deb := newDebouncer(debounceRate, eventsCh)
	go deb.Run(ctx)

	// First and last revisions because of the latestRV
	// variable being updated more frequently than the
	// debouncer puts the events at the output. Not a
	// problem, because we want the most up to date
	// revision always.
	expectedEvents := []types.APIEvent{
		{
			Name:     string(SubscriptionModeNotification),
			Revision: "0",
		},
		{
			Name:     string(SubscriptionModeNotification),
			Revision: "4",
		},
	}

	var gotEvents []types.APIEvent
	for ev := range deb.NotificationsChan() {
		fmt.Println(ev)
		gotEvents = append(gotEvents, ev)
	}
	assert.Equal(t, expectedEvents, gotEvents)
}
