package observer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type observerTest struct {
	closed bool
}

func (o *observerTest) Close() {
	o.closed = true
	fmt.Println("close observer")
}

// shutdownSequence collects the shutdown events in the order they happened.
type shutdownSequence struct {
	mu     sync.Mutex
	events []string
}

func (s *shutdownSequence) record(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)
}

func (s *shutdownSequence) recorded() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.events...)
}

// phasedObserverTest implements both contracts, so it records which phase it went through and
// when, the way a module that stops in the first phase and closes in the last one does.
type phasedObserverTest struct {
	name string
	seq  *shutdownSequence
	// stopped is released after the last observer has been stopped, which is what releases the
	// work the drain phase waits for
	stopped *sync.WaitGroup
}

func (o phasedObserverTest) Stop() {
	o.seq.record("stop:" + o.name)
	o.stopped.Done()
}

func (o phasedObserverTest) Close() {
	o.seq.record("close:" + o.name)
}

// closeOnlyObserverTest implements Observer alone, the way a component with nothing to block
// does. The pipeline must still close it.
type closeOnlyObserverTest struct {
	seq  *shutdownSequence
	name string
}

func (o closeOnlyObserverTest) Close() {
	o.seq.record("close:" + o.name)
}

// closingCounterObserverTest only counts, so the time the shutdown takes is the time the
// pipeline itself spent waiting.
type closingCounterObserverTest struct {
	closed *atomic.Int32
}

func (o closingCounterObserverTest) Close() {
	o.closed.Add(1)
}

func TestSubjectNotify(t *testing.T) {
	o := &observerTest{closed: false}
	Initialize()
	resetRunning(t)
	Attach(o)

	assert.False(t, o.closed)
	Notify()
	assert.True(t, o.closed)
}

func TestSubjectNotifyRunsTheShutdownPipeline(t *testing.T) {
	t.Run("Should stop every observer, then drain, then close every observer", func(t *testing.T) {
		const observers = 3

		withDrainTimeout(t, 5)
		Initialize()
		resetRunning(t)

		seq := &shutdownSequence{}
		var stopped sync.WaitGroup
		stopped.Add(observers)

		// work in flight, released only once every observer has been stopped: an observer
		// closed before the release means the drain phase was skipped or merged into the
		// closing one
		AddRunning()
		go func() {
			stopped.Wait()
			seq.record("drain")
			DoneRunning()
		}()

		for i := 0; i < observers; i++ {
			Attach(phasedObserverTest{name: fmt.Sprintf("module%d", i), seq: seq, stopped: &stopped})
		}

		Notify()

		assert.Equal(t,
			[]string{
				"stop:module0", "stop:module1", "stop:module2",
				"drain",
				"close:module0", "close:module1", "close:module2",
			},
			seq.recorded(),
			"the shutdown must stop everything, then drain, then close everything",
		)
	})

	t.Run("Should close an observer that does not implement Stopper", func(t *testing.T) {
		Initialize()
		resetRunning(t)

		seq := &shutdownSequence{}
		Attach(closeOnlyObserverTest{seq: seq, name: "closeOnly"})

		Notify()

		assert.Equal(t, []string{"close:closeOnly"}, seq.recorded(),
			"an observer implementing only Close must still be closed by the pipeline")
	})

	t.Run("Should wait for the running work only once", func(t *testing.T) {
		const observers = 4

		withDrainTimeout(t, 1)
		Initialize()
		resetRunning(t)

		// work nobody ever finishes, so the shutdown reaches its bound instead of returning
		// early and every extra wait costs another whole timeout
		AddRunning()
		t.Cleanup(DoneRunning)

		var closed atomic.Int32
		for i := 0; i < observers; i++ {
			Attach(closingCounterObserverTest{closed: &closed})
		}

		start := time.Now()
		Notify()
		elapsed := time.Since(start)

		assert.Equal(t, int32(observers), closed.Load(), "every observer should have been closed")
		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "the shutdown did not wait for the drain")
		assert.Less(t, elapsed, 2*time.Second,
			"the shutdown waited for the running work more than once")
	})
}
