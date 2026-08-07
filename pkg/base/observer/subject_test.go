package observer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/stretchr/testify/assert"
)

type observerTest struct {
	closed bool
}

func (o *observerTest) Close() {
	o.closed = true
	fmt.Println("close observer")
}

// orderedObserverTest records the order it was closed in, so the shutdown sequence can be
// asserted without relying on timing.
type orderedObserverTest struct {
	name  string
	mu    *sync.Mutex
	order *[]string
}

func (o orderedObserverTest) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()

	*o.order = append(*o.order, o.name)
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
	// stopped is closed after the last observer has been stopped, which is what releases the
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

// closingCounterObserverTest only counts, so the time the shutdown takes is the time the
// pipeline itself spent waiting.
type closingCounterObserverTest struct {
	closed *atomic.Int32
}

func (o closingCounterObserverTest) Close() {
	o.closed.Add(1)
}

// panickingObserverTest fails the phase it is configured to fail, the way a module with a bug
// in its shutdown does. It records itself before panicking, so the test can tell it ran.
type panickingObserverTest struct {
	name         string
	seq          *shutdownSequence
	stopped      *sync.WaitGroup
	panicOnStop  bool
	panicOnClose bool
}

func (o panickingObserverTest) Stop() {
	o.seq.record("stop:" + o.name)
	if o.stopped != nil {
		o.stopped.Done()
	}

	if o.panicOnStop {
		panic("stop exploded")
	}
}

func (o panickingObserverTest) Close() {
	o.seq.record("close:" + o.name)

	if o.panicOnClose {
		panic("close exploded")
	}
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

func TestSubjectNotify(t *testing.T) {
	o := &observerTest{closed: false}
	Initialize()
	Attach(o)

	assert.False(t, o.closed)
	Notify()
	assert.True(t, o.closed)
}

func TestSubjectNotifyRunsTheShutdownPipeline(t *testing.T) {
	t.Run("Should stop every observer, then drain, then close every observer", func(t *testing.T) {
		const observers = 3

		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 5
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		Initialize()

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

		seq := &shutdownSequence{}
		Attach(closeOnlyObserverTest{seq: seq, name: "closeOnly"})

		Notify()

		assert.Equal(t, []string{"close:closeOnly"}, seq.recorded(),
			"an observer implementing only Close must still be closed by the pipeline")
	})

	t.Run("Should wait for the running work only once", func(t *testing.T) {
		const observers = 4

		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 1
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		Initialize()

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

func TestSubjectNotifySurvivesAPanickingObserver(t *testing.T) {
	t.Run("Should keep stopping, draining and closing when a Stop panics", func(t *testing.T) {
		const stoppers = 3

		previousTimeout := config.WAIT_GROUP_TIMEOUT_SECONDS
		config.WAIT_GROUP_TIMEOUT_SECONDS = 5
		t.Cleanup(func() { config.WAIT_GROUP_TIMEOUT_SECONDS = previousTimeout })

		Initialize()

		seq := &shutdownSequence{}
		var stopped sync.WaitGroup
		stopped.Add(stoppers)

		// released only once every observer has been stopped, so a drain that never happened
		// is visible in the recorded order instead of only in the timing
		AddRunning()
		go func() {
			stopped.Wait()
			seq.record("drain")
			DoneRunning()
		}()

		Attach(phasedObserverTest{name: "module0", seq: seq, stopped: &stopped})
		Attach(panickingObserverTest{name: "broken", seq: seq, stopped: &stopped, panicOnStop: true})
		Attach(phasedObserverTest{name: "module2", seq: seq, stopped: &stopped})

		assert.NotPanics(t, Notify, "a panicking observer took the shutdown down with it")

		assert.Equal(t,
			[]string{
				"stop:module0", "stop:broken", "stop:module2",
				"drain",
				"close:module0", "close:broken", "close:module2",
			},
			seq.recorded(),
			"a panic in the stopping phase must not skip an observer, the drain or the closing phase",
		)
	})

	t.Run("Should keep closing the observers after one whose Close panics", func(t *testing.T) {
		Initialize()

		seq := &shutdownSequence{}

		Attach(closeOnlyObserverTest{seq: seq, name: "first"})
		Attach(panickingObserverTest{name: "broken", seq: seq, panicOnClose: true})
		Attach(closeOnlyObserverTest{seq: seq, name: "last"})

		assert.NotPanics(t, Notify, "a panicking observer took the shutdown down with it")

		assert.Equal(t,
			[]string{"stop:broken", "close:first", "close:broken", "close:last"},
			seq.recorded(),
			"a panic while closing must not skip the observers after it",
		)
	})
}

func TestSubjectNotifyClosesByPriority(t *testing.T) {
	t.Run("Should close the resources before what they depend on to close", func(t *testing.T) {
		Initialize()

		var mu sync.Mutex
		var order []string
		attach := func(name string, p Priority) {
			AttachWithPriority(orderedObserverTest{name: name, mu: &mu, order: &order}, p)
		}

		// attached in the order a main.go initializing monitoring before the resources it
		// records produces, which is the order the priority has to override
		attach("monitoring", PriorityLast)
		attach("sqlDB", PriorityDefault)
		attach("cacheDB", PriorityDefault)
		attach("messaging", PriorityDefault)
		Attach(orderedObserverTest{name: "unclassified", mu: &mu, order: &order})

		Notify()

		assert.Equal(t,
			[]string{"sqlDB", "cacheDB", "messaging", "unclassified", "monitoring"},
			order,
			"observers must be closed by priority, keeping the attach order inside each level",
		)
	})
}

func TestSubjectNotifyIsSafeBeforeInitialize(t *testing.T) {
	t.Run("Should do nothing when there is no subject to notify", func(t *testing.T) {
		servicesMu.Lock()
		previous := services
		services = nil
		servicesMu.Unlock()

		t.Cleanup(func() {
			servicesMu.Lock()
			services = previous
			servicesMu.Unlock()
		})

		assert.NotPanics(t, Notify)
	})
}
