package hap

import (
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type testEventWriter struct{ write func([]byte) (int, error) }

func (w *testEventWriter) Write(b []byte) (int, error) { return w.write(b) }

func TestNotifyListenerCanUnsubscribe(t *testing.T) {
	c := &Character{IID: 1, Value: true}
	var w *testEventWriter
	w = &testEventWriter{write: func(b []byte) (int, error) {
		c.RemoveListener(w)
		return len(b), nil
	}}
	c.AddListener(w)
	require.NoError(t, c.NotifyListeners(nil))
	require.Zero(t, c.ListenerCount())
}

func TestConcurrentListenerLifecycle(t *testing.T) {
	c := &Character{IID: 1, Value: true}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			// Distinct writers avoid unrelated transport concurrency.
			w := &testEventWriter{write: io.Discard.Write}
			for range 100 {
				c.AddListener(w)
				_ = c.ListenerCount()
				_ = c.NotifyListeners(nil)
				c.RemoveListener(w)
			}
		})
	}
	wg.Wait()
	require.Zero(t, c.ListenerCount())
}
