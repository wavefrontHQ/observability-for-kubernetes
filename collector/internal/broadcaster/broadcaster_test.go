package broadcaster_test

import (
	"io"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/broadcaster"
)

const DefaultPublishTimeout = 5 * time.Second

func TestBroadcaster(t *testing.T) {
	restoreOriginalLogger := setupLoggerForTesting()
	t.Cleanup(restoreOriginalLogger)

	t.Run("broadcasts a line to single subscriber", func(t *testing.T) {
		b := broadcaster.New[string]()
		lines, unsubscribe := b.Subscribe()
		defer unsubscribe()

		go b.Publish(DefaultPublishTimeout, "some line")

		actualLine := requireReceiveWithin(t, time.Second, lines)
		require.Equal(t, "some line", actualLine)
	})

	t.Run("broadcasts a line to multiple subscribers", func(t *testing.T) {
		b := broadcaster.New[string]()
		linesA, unsubscribeA := b.Subscribe()
		defer unsubscribeA()
		linesB, unsubscribeB := b.Subscribe()
		defer unsubscribeB()

		go b.Publish(DefaultPublishTimeout, "some line")

		actualLine := requireReceiveWithin(t, DefaultPublishTimeout, linesA)
		require.Equal(t, "some line", actualLine)

		actualLine = requireReceiveWithin(t, DefaultPublishTimeout, linesB)
		require.Equal(t, "some line", actualLine)
	})

	t.Run("does not send to clients which exceed the publish timeout", func(t *testing.T) {
		b := broadcaster.New[string]()
		lines, unsubscribe := b.Subscribe()
		defer unsubscribe()

		go b.Publish(time.Millisecond, "some line")

		time.Sleep(10 * time.Millisecond)

		go b.Publish(DefaultPublishTimeout, "another line")

		actualLine := requireReceiveWithin(t, DefaultPublishTimeout, lines)
		require.Equal(t, "another line", actualLine)
	})

	t.Run("unsubscribing causes no more lines to be received on that subscription", func(t *testing.T) {
		b := broadcaster.New[string]()
		lines, unsubscribe := b.Subscribe()
		unsubscribe()

		go b.Publish(DefaultPublishTimeout, "some line")

		actualLine := requireReceiveWithin(t, DefaultPublishTimeout, lines)
		require.Equal(t, "", actualLine)
	})
}

func setupLoggerForTesting() func() {
	original := log.StandardLogger().Out
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(original)
	}
}

func requireReceiveWithin[M any](t *testing.T, timeout time.Duration, messages <-chan M) M {
	t.Helper()
	var message M
	select {
	case message = <-messages:
	case <-time.After(timeout):
		t.Fatalf("did not receive message within %s", timeout.String())
	}
	return message
}
