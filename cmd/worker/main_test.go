package main

import (
	"context"
	"testing"
	"time"
)

func TestStartPollLoopWaitsForInFlightProcessBeforeClosingDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticks := make(chan time.Time, 1)
	started := make(chan struct{})
	release := make(chan struct{})

	done := startPollLoop(ctx, ticks, func() {
		close(started)
		<-release
	})

	ticks <- time.Now()
	<-started
	cancel()

	select {
	case <-done:
		t.Fatal("done closed before in-flight process finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("done did not close after process finished")
	}
}

func TestStartPollLoopClosesWhenCanceledIdle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)

	done := startPollLoop(ctx, ticks, func() {
		t.Fatal("process should not run while idle")
	})

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("done did not close after idle cancel")
	}
}
