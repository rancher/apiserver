package subscribe

import (
	"context"
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
		gotEvents = append(gotEvents, ev)
	}
	assert.Equal(t, expectedEvents, gotEvents)
}

func TestDebouncerCanceled(t *testing.T) {
	in := make(chan types.APIEvent)
	debouncer := newDebouncer(100*time.Millisecond, in)
	out := debouncer.NotificationsChan()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // already canceled

	runFinished := make(chan struct{})
	go func() {
		debouncer.Run(ctx)
		close(runFinished)
	}()

	select {
	case <-runFinished:
		t.Error("debouncer finished before original channel was closed")
	case <-out:
		t.Error("debouncer channel closed before the original channel")
	case <-time.After(200 * time.Millisecond):
		// all good
	}

	// Writing to in is not blocked, since it must be drained
	for range 3 {
		select {
		case in <- types.APIEvent{}:
		case <-time.After(10 * time.Millisecond):
			t.Error("input channel is not being drained")
		}
	}
	close(in)

	select {
	case <-time.After(200 * time.Millisecond):
		t.Error("output channel was not closed")
	case _, ok := <-runFinished:
		if ok {
			t.Error("unexpected item received while waiting for the output channel to be closed")
		}
	}

	select {
	case <-time.After(200 * time.Millisecond):
		t.Error("debouncer goroutine didn't finish")
	case <-runFinished:
	}
}
