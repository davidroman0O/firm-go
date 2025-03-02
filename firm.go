package firm

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/sasha-s/go-deadlock"
)

// Initialize deadlock detection
func init() {
	deadlock.Opts.DeadlockTimeout = time.Second * 2
	deadlock.Opts.OnPotentialDeadlock = func() {
		fmt.Println("Potential deadlock detected")
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, true)
		fmt.Printf("Stack trace:\n%s\n", string(buf[:n]))
	}
}

// CleanUp is a function that performs cleanup operations
type CleanUp func()

// Reactive is the interface implemented by all reactive objects
type Reactive interface {
	getNode() *node
}

// node represents a node in the dependency graph
type node struct {
	mu         deadlock.Mutex
	dependents []*node
	runEffect  func()
	disposed   bool
	owner      *Owner
}

func (n *node) addDependent(dep *node) {
	if n.disposed {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check if already exists
	for _, existing := range n.dependents {
		if existing == dep {
			return
		}
	}

	n.dependents = append(n.dependents, dep)
}

func (n *node) removeDependent(dep *node) {
	if n.disposed {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	for i, existing := range n.dependents {
		if existing == dep {
			// Remove by swapping with last element
			lastIdx := len(n.dependents) - 1
			n.dependents[i] = n.dependents[lastIdx]
			n.dependents = n.dependents[:lastIdx]
			return
		}
	}
}

func (n *node) notify() {
	if n.disposed {
		return
	}

	// If batch mode is active, queue the notification
	n.mu.Lock()
	// Check if owner exists and get a local reference
	owner := n.owner
	n.mu.Unlock()

	if owner != nil {
		owner.mu.Lock()
		batchActive := owner.batchActive
		owner.mu.Unlock()

		if batchActive {
			owner.queueNotification(n)
			return
		}
	}

	// Otherwise process directly
	processNodeNotify(n)
}

func (n *node) dispose() {
	n.mu.Lock()
	n.disposed = true
	n.dependents = nil
	n.runEffect = nil
	n.mu.Unlock()
}

// Owner manages a reactive system
type Owner struct {
	mu            deadlock.Mutex
	nodes         []*node
	currentEffect *node
	tracking      bool
	batchActive   bool
	batchQueue    []*node
	batchSet      map[*node]struct{}
	cleanups      []CleanUp
}

// NewOwner creates a new owner
func NewOwner() *Owner {
	return &Owner{
		nodes:      make([]*node, 0),
		batchSet:   make(map[*node]struct{}),
		batchQueue: make([]*node, 0),
		cleanups:   make([]CleanUp, 0),
	}
}

// Root creates a reactive system
func Root(fn func(r *Owner) CleanUp) CleanUp {
	owner := NewOwner()
	cleanup := fn(owner)

	return func() {
		// First run the user-provided cleanup
		if cleanup != nil {
			cleanup()
		}

		// Then clean up the owner
		owner.dispose()
	}
}

func createNode(owner *Owner) *node {
	owner.mu.Lock()
	defer owner.mu.Unlock()

	n := &node{
		dependents: make([]*node, 0),
		owner:      owner,
	}

	owner.nodes = append(owner.nodes, n)
	return n
}

func (owner *Owner) queueNotification(n *node) {
	owner.mu.Lock()
	defer owner.mu.Unlock()

	// Only add if not already in queue
	if _, exists := owner.batchSet[n]; !exists {
		owner.batchQueue = append(owner.batchQueue, n)
		owner.batchSet[n] = struct{}{}
	}
}

func (owner *Owner) processBatchQueue() {
	owner.mu.Lock()
	queue := make([]*node, len(owner.batchQueue))
	copy(queue, owner.batchQueue)
	owner.batchQueue = owner.batchQueue[:0]
	owner.batchSet = make(map[*node]struct{})
	owner.mu.Unlock()

	for _, n := range queue {
		if !n.disposed {
			n.notify()
		}
	}
}

func (owner *Owner) trackDependency(dep *node) {
	owner.mu.Lock()
	defer owner.mu.Unlock()

	if !owner.tracking || owner.currentEffect == nil || dep == nil {
		return
	}

	dep.addDependent(owner.currentEffect)
}

func (owner *Owner) beginTracking(effect *node) {
	owner.mu.Lock()
	defer owner.mu.Unlock()

	owner.currentEffect = effect
	owner.tracking = true
}

func (owner *Owner) endTracking() {
	owner.mu.Lock()
	defer owner.mu.Unlock()

	owner.currentEffect = nil
	owner.tracking = false
}

// dispose runs all cleanup functions for this owner
func (o *Owner) dispose() {
	o.mu.Lock()
	// Make a local copy of cleanups
	cleanups := make([]CleanUp, len(o.cleanups))
	copy(cleanups, o.cleanups)
	// Clear cleanups first to prevent re-running
	o.cleanups = nil
	o.mu.Unlock()

	// Run cleanups in reverse order (LIFO)
	for i := len(cleanups) - 1; i >= 0; i-- {
		if cleanups[i] != nil {
			cleanups[i]()
		}
	}
}

// signalImpl holds a reactive value (private implementation)
type signalImpl[T any] struct {
	value T
	owner *Owner
	node  *node
	mu    deadlock.RWMutex
}

// Signal creates a reactive value
func Signal[T any](owner *Owner, initialValue T) *signalImpl[T] {
	return &signalImpl[T]{
		value: initialValue,
		owner: owner,
		node:  createNode(owner),
	}
}

func (s *signalImpl[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Track as dependency if in tracking mode
	if s.owner != nil {
		s.owner.trackDependency(s.node)
	}

	return s.value
}

func (s *signalImpl[T]) Set(value T) {
	s.mu.Lock()

	// Skip update if value is equal
	if reflect.DeepEqual(s.value, value) {
		s.mu.Unlock()
		return
	}

	s.value = value
	s.mu.Unlock()

	// Notify dependents
	s.node.notify()
}

func (s *signalImpl[T]) Peek() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *signalImpl[T]) Update(fn func(T) T) {
	s.mu.Lock()
	oldValue := s.value
	newValue := fn(oldValue)
	s.value = newValue
	changed := !reflect.DeepEqual(oldValue, newValue)
	s.mu.Unlock()

	// Only notify if the value actually changed
	if changed {
		// Use batch to handle concurrent notifications more reliably
		if s.owner != nil {
			Batch(s.owner, func() {
				s.node.notify()
			})
		} else {
			s.node.notify()
		}
	}
}

func (s *signalImpl[T]) getNode() *node {
	return s.node
}

// Effect creates a side effect that runs when dependencies change
func Effect(owner *Owner, fn func() CleanUp, deps []Reactive) {
	// Create effect node
	effectNode := createNode(owner)

	// Use mutex to protect access to cleanup and dependencies
	var effectMu deadlock.Mutex
	var cleanup CleanUp

	// Dependencies of this effect
	dependencies := make([]*node, 0, len(deps))

	// Set up the effect function
	effectNode.runEffect = func() {
		// Run cleanup from previous execution
		effectMu.Lock()
		if cleanup != nil {
			localCleanup := cleanup // Make a local copy to run outside the lock
			cleanup = nil
			effectMu.Unlock()
			localCleanup()
		} else {
			effectMu.Unlock()
		}

		// Remove existing dependencies - need lock for this
		effectMu.Lock()
		depsToRemove := make([]*node, len(dependencies))
		copy(depsToRemove, dependencies)
		dependencies = dependencies[:0]
		effectMu.Unlock()

		for _, dep := range depsToRemove {
			dep.removeDependent(effectNode)
		}

		// Track new dependencies if none specified
		var newCleanup CleanUp
		if len(deps) == 0 {
			owner.beginTracking(effectNode)
			newCleanup = fn()
			owner.endTracking()
		} else {
			// Otherwise use explicit dependencies
			newCleanup = fn()

			// Add dependencies - safely add to dependencies
			effectMu.Lock()
			for _, dep := range deps {
				node := dep.getNode()
				node.addDependent(effectNode)
				dependencies = append(dependencies, node)
			}
			effectMu.Unlock()
		}

		// Store new cleanup function with mutex protection
		effectMu.Lock()
		cleanup = newCleanup
		effectMu.Unlock()
	}

	// Run initially
	effectNode.runEffect()

	// Register cleanup with thread-safe access
	owner.mu.Lock()
	owner.cleanups = append(owner.cleanups, func() {
		effectMu.Lock()
		localCleanup := cleanup
		cleanup = nil

		// Get dependencies to remove under lock
		depsToRemove := make([]*node, len(dependencies))
		copy(depsToRemove, dependencies)
		dependencies = nil
		effectMu.Unlock()

		if localCleanup != nil {
			localCleanup()
		}

		// Remove dependencies
		for _, dep := range depsToRemove {
			dep.removeDependent(effectNode)
		}

		effectNode.dispose()
	})
	owner.mu.Unlock()
}

// Memo creates a computed value
func Memo[T any](owner *Owner, compute func() T, deps []Reactive) *signalImpl[T] {
	signal := Signal(owner, *new(T))

	Effect(owner, func() CleanUp {
		signal.Set(compute())
		return func() {}
	}, deps)

	// Initial computation
	signal.Set(compute())

	return signal
}

// Untrack prevents dependency tracking
func Untrack[T any](owner *Owner, fn func() T) T {
	owner.mu.Lock()
	oldTracking := owner.tracking
	oldEffect := owner.currentEffect
	owner.tracking = false
	owner.currentEffect = nil
	owner.mu.Unlock()

	result := fn()

	owner.mu.Lock()
	owner.tracking = oldTracking
	owner.currentEffect = oldEffect
	owner.mu.Unlock()

	return result
}

// Global mutex to synchronize batch operations
var batchMutex deadlock.Mutex

// Batch runs multiple updates as a batch
func Batch(owner *Owner, fn func()) {
	// Save previous batch state
	owner.mu.Lock()
	oldBatch := owner.batchActive
	owner.batchActive = true
	owner.mu.Unlock()

	// Run the batch function
	fn()

	// Restore previous batch state
	owner.mu.Lock()
	wasActive := owner.batchActive
	owner.batchActive = oldBatch

	// If we're returning to non-batch mode, process queue
	if wasActive && !oldBatch {
		// Make a local copy of the queue
		queue := make([]*node, len(owner.batchQueue))
		copy(queue, owner.batchQueue)

		// Clear the queue
		owner.batchQueue = owner.batchQueue[:0]
		owner.batchSet = make(map[*node]struct{})
		owner.mu.Unlock()

		// Process queue outside the lock
		for _, n := range queue {
			if n != nil && !n.disposed {
				processNode(n)
			}
		}
	} else {
		owner.mu.Unlock()
	}
}

// Helper to process nodes without locking issues
func processNode(n *node) {
	// Skip if node is disposed
	if n.disposed {
		return
	}

	// Get snapshot of dependents
	n.mu.Lock()
	deps := make([]*node, len(n.dependents))
	copy(deps, n.dependents)
	n.mu.Unlock()

	// Set to track already processed effects
	processed := make(map[*node]struct{})

	// Process each dependent
	for _, dep := range deps {
		if dep == nil || dep.disposed || dep.runEffect == nil {
			continue
		}

		if _, alreadyProcessed := processed[dep]; alreadyProcessed {
			continue
		}

		processed[dep] = struct{}{}
		dep.runEffect()
	}
}

// Helper function to process node notifications
func processNodeNotify(n *node) {
	// Make a copy of dependents under lock and check disposed state
	n.mu.Lock()
	if n.disposed {
		n.mu.Unlock()
		return
	}
	deps := make([]*node, len(n.dependents))
	copy(deps, n.dependents)
	n.mu.Unlock()

	// Process dependents, carefully checking their state
	for _, dep := range deps {
		if dep == nil {
			continue
		}

		// Get runEffect function safely
		dep.mu.Lock()
		runEffect := dep.runEffect
		isDisposed := dep.disposed
		dep.mu.Unlock()

		if runEffect != nil && !isDisposed {
			runEffect()
		}
	}
}

// Context provides value propagation through the reactive tree
type Context[T any] struct {
	value *signalImpl[T]
	owner *Owner
	node  *node
}

// NewContext creates a new context with a default value
func NewContext[T any](owner *Owner, defaultValue T) *Context[T] {
	context := &Context[T]{
		value: Signal(owner, defaultValue),
		owner: owner,
		node:  createNode(owner),
	}

	// Make the context's node reflect the value's node
	Effect(owner, func() CleanUp {
		// Get value to track dependency
		context.value.Get()
		context.node.notify()
		return nil
	}, []Reactive{context.value})

	return context
}

// Use retrieves the current context value
func (c *Context[T]) Use() T {
	// Track this context as a dependency
	if c.owner != nil {
		c.owner.trackDependency(c.node)
	}
	return c.value.Get()
}

// Set changes the context value
func (c *Context[T]) Set(value T) {
	c.value.Set(value)
}

// getNode implements the Reactive interface
func (c *Context[T]) getNode() *node {
	return c.node
}

// Match runs code only when context exactly matches a value
func (c *Context[T]) Match(
	owner *Owner,
	value T,
	fn func(childOwner *Owner) CleanUp,
) CleanUp {
	childOwner := NewOwner()

	var childCleanup CleanUp
	var isActive bool = false

	// Effect that watches the context value
	Effect(owner, func() CleanUp {
		currentValue := c.Use()
		matched := reflect.DeepEqual(currentValue, value)

		// If match state changed
		if matched != isActive {
			// If now matched, create the child scope
			if matched {
				childCleanup = fn(childOwner)
				isActive = true
			} else if childCleanup != nil {
				// If no longer matched, clean up the child scope
				childCleanup()
				childCleanup = nil
				isActive = false
			}
		}

		return func() {
			if childCleanup != nil {
				childCleanup()
				childCleanup = nil
			}
		}
	}, []Reactive{c})

	// Return a cleanup function
	return func() {
		if childCleanup != nil {
			childCleanup()
		}
		childOwner.dispose()
	}
}

// When runs the provided function only when the context value matches the condition
func (c *Context[T]) When(
	owner *Owner,
	matcher func(T) bool,
	fn func(childOwner *Owner) CleanUp,
) CleanUp {
	childOwner := NewOwner()

	var childCleanup CleanUp
	var isActive bool = false

	// Effect that watches the context value
	Effect(owner, func() CleanUp {
		currentValue := c.Use()
		matched := matcher(currentValue)

		// If match state changed
		if matched != isActive {
			// If now matched, create the child scope
			if matched {
				childCleanup = fn(childOwner)
				isActive = true
			} else if childCleanup != nil {
				// If no longer matched, clean up the child scope
				childCleanup()
				childCleanup = nil
				isActive = false
			}
		}

		return func() {
			if childCleanup != nil {
				childCleanup()
				childCleanup = nil
			}
		}
	}, []Reactive{c})

	// Return a cleanup function
	return func() {
		if childCleanup != nil {
			childCleanup()
		}
		childOwner.dispose()
	}
}

// resourceImpl represents async data (private implementation)
type resourceImpl[T any] struct {
	owner   *Owner
	loading *signalImpl[bool]
	data    *signalImpl[T]
	error   *signalImpl[error]
	refetch func()
	node    *node
	mu      deadlock.Mutex
}

// Resource creates a resource with async fetcher
func Resource[T any](
	owner *Owner,
	fetcher func() (T, error),
) *resourceImpl[T] {
	loading := Signal(owner, true)
	data := Signal(owner, *new(T))
	err := Signal(owner, error(nil))

	// Node for the resource
	resourceNode := createNode(owner)

	// Use WaitGroup for synchronization
	var fetchWg deadlock.WaitGroup
	var fetchMu deadlock.Mutex
	var active bool = true

	resource := &resourceImpl[T]{
		owner:   owner,
		loading: loading,
		data:    data,
		error:   err,
		node:    resourceNode,
	}

	// Create refetch function with better synchronization
	resource.refetch = func() {
		fetchMu.Lock()
		if !active {
			fetchMu.Unlock()
			return
		}
		loading.Set(true)
		fetchWg.Add(1)
		fetchMu.Unlock()

		go func() {
			defer fetchWg.Done()

			result, fetchErr := fetcher()

			fetchMu.Lock()
			if !active {
				fetchMu.Unlock()
				return
			}

			Batch(owner, func() {
				if fetchErr != nil {
					err.Set(fetchErr)
				} else {
					data.Set(result)
				}
				loading.Set(false)
				resourceNode.notify()
			})
			fetchMu.Unlock()
		}()
	}

	// Initial fetch
	resource.refetch()

	// Register cleanup that waits for ongoing fetches
	owner.mu.Lock()
	owner.cleanups = append(owner.cleanups, func() {
		fetchMu.Lock()
		active = false
		fetchMu.Unlock()

		// Wait for any ongoing fetches to complete
		fetchWg.Wait()
	})
	owner.mu.Unlock()

	return resource
}

func (r *resourceImpl[T]) Loading() bool {
	r.owner.trackDependency(r.loading.node)
	return r.loading.Get()
}

func (r *resourceImpl[T]) Data() T {
	r.owner.trackDependency(r.data.node)
	return r.data.Get()
}

func (r *resourceImpl[T]) Error() error {
	r.owner.trackDependency(r.error.node)
	return r.error.Get()
}

func (r *resourceImpl[T]) Refetch() {
	r.refetch()
}

func (r *resourceImpl[T]) OnLoad(fn func(data T, err error)) {
	Effect(r.owner, func() CleanUp {
		if !r.Loading() {
			fn(r.Data(), r.Error())
		}
		return func() {}
	}, []Reactive{r})
}

func (r *resourceImpl[T]) getNode() *node {
	return r.node
}

// Defer creates a signal that updates after a delay
func Defer[T any](owner *Owner, source *signalImpl[T], timeoutMs int) *signalImpl[T] {
	result := Signal(owner, source.Peek())

	var timer *time.Timer

	Effect(owner, func() CleanUp {
		value := source.Get()

		// Cancel existing timer
		if timer != nil {
			timer.Stop()
		}

		// Create new timer
		timer = time.AfterFunc(time.Duration(timeoutMs)*time.Millisecond, func() {
			result.Set(value)
		})

		return func() {
			if timer != nil {
				timer.Stop()
				timer = nil
			}
		}
	}, []Reactive{source})

	return result
}

// Map transforms values with a mapping function
func Map[T any, R any](
	owner *Owner,
	source *signalImpl[[]T],
	mapFn func(item T, index int) R,
) *signalImpl[[]R] {
	return Memo(owner, func() []R {
		items := source.Get()
		result := make([]R, len(items))
		for i, item := range items {
			result[i] = mapFn(item, i)
		}
		return result
	}, []Reactive{source})
}

// DerivedSignal creates a signal that's derived from another
func DerivedSignal[T any, R any](
	owner *Owner,
	source *signalImpl[T],
	get func(T) R,
	set func(T, R) T,
) *signalImpl[R] {
	// Create derived signal
	derived := Signal(owner, get(source.Peek()))

	// Set up bidirectional binding
	var derivedCleanup CleanUp

	// Track changes from source to derived
	Effect(owner, func() CleanUp {
		// Update derived when source changes
		derived.Set(get(source.Get()))

		// Set up reverse tracking (derived to source)
		derivedCleanup = func() CleanUp {
			Effect(owner, func() CleanUp {
				// When derived changes, update source
				newSource := set(source.Peek(), derived.Get())

				// Only update if changed to avoid cycles
				if !reflect.DeepEqual(source.Peek(), newSource) {
					Untrack(owner, func() T {
						source.Set(newSource)
						return newSource
					})
				}
				return func() {}
			}, []Reactive{derived})

			return func() {}
		}()

		return func() {
			if derivedCleanup != nil {
				derivedCleanup()
				derivedCleanup = nil
			}
		}
	}, []Reactive{source})

	return derived
}

// Computed represents a signal backed by a computation function
type Computed[T any] struct {
	value    *signalImpl[T]
	compute  func() T
	owner    *Owner
	lastHash string // For detecting changes
	mu       deadlock.Mutex
}

// NewComputed creates a signal that gets its value from a compute function
func NewComputed[T any](owner *Owner, compute func() T) *Computed[T] {
	// Compute initial value
	initial := compute()
	computed := &Computed[T]{
		value:   Signal(owner, initial),
		compute: compute,
		owner:   owner,
	}

	// Store hash of initial value
	computed.updateHash(initial)

	return computed
}

// Get returns the current value
func (c *Computed[T]) Get() T {
	return c.value.Get()
}

// Recompute executes the compute function and updates value if changed
func (c *Computed[T]) Recompute() bool {
	// Safely compute new value
	c.mu.Lock()
	newValue := c.compute()
	changed := c.hasChanged(newValue)

	if changed {
		c.updateHash(newValue)
		// Keep a copy of the value to set outside the lock
		valueToSet := newValue
		c.mu.Unlock()

		// Set the new value after releasing our lock
		c.value.Set(valueToSet)
		return true
	}

	c.mu.Unlock()
	return false
}

// updateHash stores a hash of the current value for change detection
func (c *Computed[T]) updateHash(value T) {
	hash := fmt.Sprintf("%v", value)
	c.lastHash = hash
}

// hasChanged checks if value has changed from last value
func (c *Computed[T]) hasChanged(value T) bool {
	hash := fmt.Sprintf("%v", value)
	return hash != c.lastHash
}

// getNode implements the Reactive interface
func (c *Computed[T]) getNode() *node {
	return c.value.getNode()
}

// Polling is a computed signal that refreshes on a timer
type Polling[T any] struct {
	*Computed[T]
	ticker   *time.Ticker
	stopChan chan struct{}
	interval time.Duration
	mu       deadlock.Mutex
	active   bool
}

// NewPolling creates a computed signal that recomputes on an interval
func NewPolling[T any](
	owner *Owner,
	compute func() T,
	interval time.Duration,
) *Polling[T] {
	// Create base computed signal
	computed := NewComputed(owner, compute)

	// Create mutex-protected state
	var mu deadlock.Mutex
	active := true

	// Create polling signal
	polling := &Polling[T]{
		Computed: computed,
		interval: interval,
		mu:       deadlock.Mutex{},
		active:   true,
	}

	// Create channels under lock
	mu.Lock()
	ticker := time.NewTicker(interval)
	stopChan := make(chan struct{})
	polling.ticker = ticker
	polling.stopChan = stopChan
	mu.Unlock()

	// For cleanup synchronization
	var wg deadlock.WaitGroup
	wg.Add(1)

	// Start refresh loop in goroutine
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ticker.C:
				mu.Lock()
				if !active {
					mu.Unlock()
					return
				}

				// Get a local reference to the computed
				comp := polling.Computed
				mu.Unlock()

				if comp != nil {
					// Need global mutex to safely batch
					batchMutex.Lock()
					owner.mu.Lock()
					oldBatch := owner.batchActive
					owner.batchActive = true
					owner.mu.Unlock()

					// Try to recompute
					comp.Recompute()

					// Restore batch state and process if needed
					owner.mu.Lock()
					owner.batchActive = oldBatch
					if !oldBatch {
						queue := make([]*node, len(owner.batchQueue))
						copy(queue, owner.batchQueue)
						owner.batchQueue = owner.batchQueue[:0]
						owner.batchSet = make(map[*node]struct{})
						owner.mu.Unlock()
						batchMutex.Unlock()

						for _, n := range queue {
							if n != nil && !n.disposed {
								processNodeNotify(n)
							}
						}
					} else {
						owner.mu.Unlock()
						batchMutex.Unlock()
					}
				}

			case <-stopChan:
				return
			}
		}
	}()

	// Register cleanup to stop the goroutine and wait for it
	owner.mu.Lock()
	owner.cleanups = append(owner.cleanups, func() {
		mu.Lock()
		active = false
		if ticker != nil {
			ticker.Stop()
		}
		close(stopChan)
		mu.Unlock()

		// Wait for goroutine to finish
		wg.Wait()
	})
	owner.mu.Unlock()

	return polling
}

// SetInterval changes the refresh interval
func (p *Polling[T]) SetInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interval = interval
	if p.ticker != nil && p.active {
		p.ticker.Reset(interval)
	}
}

// Pause temporarily stops auto-refreshing
func (p *Polling[T]) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ticker != nil && p.active {
		p.ticker.Stop()
	}
}

// Resume restarts auto-refreshing
func (p *Polling[T]) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ticker != nil && p.active {
		p.ticker.Reset(p.interval)
	}
}
