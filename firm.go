package firm

import (
	"fmt"
	"reflect"
	"sync"
)

// Context represents a reactive execution context that can be used to
// track and manage dependencies, similar to Solid.js createRoot
type Context struct {
	disposables []func()
	parent      *Context
}

// Memo represents a memoized value (optimized computed value)
type Memo[T any] struct {
	Computed[T]
	equals func(T, T) bool
}

// Resource represents an async resource with loading/error states
type Resource[T any] struct {
	signal   Signal[ResourceState[T]]
	fetcher  func() (T, error)
	refetch  func()
	disposed bool
}

// ResourceState represents the different states of a resource
type ResourceState[T any] struct {
	Loading bool
	Error   error
	Data    T
}

// Signal represents a reactive value that can be observed for changes
type Signal[T any] struct {
	value      T
	listeners  []func(T)
	mutex      sync.RWMutex
	equalityFn func(T, T) bool
	context    *Context
}

// Computed represents a derived value that depends on one or more signals
type Computed[T any] struct {
	signal    Signal[T]
	compute   func() T
	deps      []any // dependencies that this computed value listens to
	cleanup   func()
	isStale   bool
	untracked bool
	context   *Context
}

// ForceComputation forces a recomputation of the computed value
func (c *Computed[T]) ForceComputation() {
	c.isStale = true
	c.Get()
}

// Effect represents a side effect that runs when its dependencies change
type Effect struct {
	execute  func()
	deps     []any
	cleanup  func()
	disposed bool
	context  *Context
}

// Store represents a reactive store (similar to Solid's createStore)
type Store[T any] struct {
	state      Signal[T]
	middleware []func(path []string, value any, oldValue any) any
}

// Current tracking context for automatic dependency tracking
var (
	activeContext    *Context
	currentObserver  any
	runningEffects   []Effect
	defaultContext   *Context
	contextStack     []*Context
	disposeOnUnmount []func()
)

// Batch functionality to batch updates
var (
	batchQueue     []func()
	isBatchingFlag bool
	batchMutex     sync.Mutex
)

// Lifecycle hooks
var (
	onMountHooks   []func()
	onCleanupHooks []func()
)

// Initialize the default context
func init() {
	defaultContext = &Context{
		disposables: make([]func(), 0),
		parent:      nil,
	}
	activeContext = defaultContext
}

// CreateRoot creates a new root context with isolated reactivity
func CreateRoot(fn func()) func() {
	parentContext := activeContext
	newContext := &Context{
		disposables: make([]func(), 0),
		parent:      parentContext,
	}

	// Set the new context as active
	contextStack = append(contextStack, activeContext)
	activeContext = newContext

	// Run the function in the new context
	fn()

	// Restore the previous context
	activeContext = contextStack[len(contextStack)-1]
	contextStack = contextStack[:len(contextStack)-1]

	// Return a dispose function
	return func() {
		for _, dispose := range newContext.disposables {
			dispose()
		}
		newContext.disposables = nil
	}
}

// NewSignal creates a new signal with the provided initial value
func NewSignal[T any](initialValue T) *Signal[T] {
	s := &Signal[T]{
		value:     initialValue,
		listeners: make([]func(T), 0),
		equalityFn: func(a, b T) bool {
			return reflect.DeepEqual(a, b)
		},
		context: activeContext,
	}

	if activeContext != nil {
		activeContext.disposables = append(activeContext.disposables, func() {
			s.listeners = nil
		})
	}

	return s
}

// SetEqualityFn sets a custom equality function for the signal
func (s *Signal[T]) SetEqualityFn(fn func(T, T) bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.equalityFn = fn
}

// Get returns the current value of the signal and tracks it as a dependency
func (s *Signal[T]) Get() T {
	s.mutex.RLock()
	value := s.value
	s.mutex.RUnlock()

	// Track this signal as a dependency of the current observer
	if currentObserver != nil {
		switch observer := currentObserver.(type) {
		case *Computed[T]:
			// Add a subscription to this signal
			unsubscribe := s.Subscribe(func(newValue T) {
				if observer.isStale {
					return
				}
				observer.isStale = true

				// If we're not within a batch, recompute immediately
				if !isBatching() {
					prevObserver := currentObserver
					currentObserver = observer
					observer.signal.Set(observer.compute())
					currentObserver = prevObserver
					observer.isStale = false
				}
			})

			// Store the unsubscribe function to clean up later
			if observer.cleanup == nil {
				observer.cleanup = unsubscribe
			} else {
				oldCleanup := observer.cleanup
				observer.cleanup = func() {
					oldCleanup()
					unsubscribe()
				}
			}
		case *Effect:
			// Add a subscription to this signal
			unsubscribe := s.Subscribe(func(newValue T) {
				if !observer.disposed {
					execute(observer)
				}
			})

			// Store the unsubscribe function to clean up later
			if observer.cleanup == nil {
				observer.cleanup = unsubscribe
			} else {
				oldCleanup := observer.cleanup
				observer.cleanup = func() {
					oldCleanup()
					unsubscribe()
				}
			}
		default:
			// Handle other types of observers
			// This is mainly to support generic type parameters
			switch any(observer).(type) {
			case *Computed[any]:
				comp := any(observer).(*Computed[any])
				unsubscribe := s.Subscribe(func(newValue T) {
					if comp.isStale {
						return
					}
					comp.isStale = true

					// If we're not within a batch, recompute immediately
					if !isBatching() {
						prevObserver := currentObserver
						currentObserver = comp
						comp.signal.Set(comp.compute())
						currentObserver = prevObserver
						comp.isStale = false
					}
				})

				if comp.cleanup == nil {
					comp.cleanup = unsubscribe
				} else {
					oldCleanup := comp.cleanup
					comp.cleanup = func() {
						oldCleanup()
						unsubscribe()
					}
				}
			case *Effect:
				eff := any(observer).(*Effect)
				unsubscribe := s.Subscribe(func(newValue T) {
					if !eff.disposed {
						execute(eff)
					}
				})

				if eff.cleanup == nil {
					eff.cleanup = unsubscribe
				} else {
					oldCleanup := eff.cleanup
					eff.cleanup = func() {
						oldCleanup()
						unsubscribe()
					}
				}
			}
		}
	}

	return value
}

// GetUntracked returns the current value without tracking it as a dependency
func (s *Signal[T]) GetUntracked() T {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.value
}

// Set updates the signal's value and notifies listeners if the value changed
func (s *Signal[T]) Set(newValue T) {
	var valueChanged bool
	var listeners []func(T)

	s.mutex.Lock()
	// Compare old and new values using the equality function
	oldValue := s.value
	valueChanged = !s.equalityFn(oldValue, newValue)

	if valueChanged {
		s.value = newValue
		// Make a copy of listeners to avoid holding the lock during callbacks
		listeners = make([]func(T), len(s.listeners))
		copy(listeners, s.listeners)
	}
	s.mutex.Unlock()

	// Notify listeners outside the lock
	if valueChanged {
		if isBatching() {
			batchUpdate(func() {
				for _, listener := range listeners {
					listener(newValue)
				}
			})
		} else {
			for _, listener := range listeners {
				listener(newValue)
			}
		}
	}
}

// Subscribe adds a listener to the signal
func (s *Signal[T]) Subscribe(listener func(T)) func() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.listeners = append(s.listeners, listener)

	// Return a function to remove this listener
	return func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		for i, l := range s.listeners {
			if fmt.Sprintf("%p", l) == fmt.Sprintf("%p", listener) {
				s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
				break
			}
		}
	}
}

// Update allows updating the signal based on its current value
func (s *Signal[T]) Update(fn func(T) T) {
	s.Set(fn(s.Get()))
}

// Peek gets the signal's value without creating a dependency
func (s *Signal[T]) Peek() T {
	return s.GetUntracked()
}

// NewComputed creates a computed signal derived from other signals
func NewComputed[T any](compute func() T) *Computed[T] {
	// Create a type-specific equality function that uses reflect.DeepEqual internally
	equalityFn := func(a, b T) bool {
		return reflect.DeepEqual(a, b)
	}

	comp := &Computed[T]{
		signal: Signal[T]{
			value:      *new(T), // Initialize with zero value
			listeners:  make([]func(T), 0),
			equalityFn: equalityFn,
			context:    activeContext,
		},
		compute: compute,
		deps:    make([]any, 0),
		context: activeContext,
		isStale: true, // Start as stale to ensure initial computation
	}

	// Register with context if needed
	if activeContext != nil {
		activeContext.disposables = append(activeContext.disposables, func() {
			if comp.cleanup != nil {
				comp.cleanup()
				comp.cleanup = nil
			}
		})
	}

	return comp
}

func setupComputed[T any](comp *Computed[T]) {
	// Create a subscriber for each dependency
	comp.cleanup = func() {
		for _, dep := range comp.deps {
			switch d := dep.(type) {
			case *Signal[any]:
				d.Subscribe(func(any) {
					if !comp.isStale {
						comp.isStale = true

						// If we're not within a batch, recompute immediately
						if !isBatching() {
							prevObserver := currentObserver
							currentObserver = comp
							comp.signal.Set(comp.compute())
							currentObserver = prevObserver
							comp.isStale = false
						}
					}
				})
			}
		}
	}

	comp.cleanup()
}

// Get returns the computed value and tracks it as a dependency
func (c *Computed[T]) Get() T {
	// Re-compute if stale
	if c.isStale {
		prevObserver := currentObserver
		currentObserver = c
		value := c.compute()
		c.signal.Set(value)
		currentObserver = prevObserver
		c.isStale = false
	}

	// Track this computed as a dependency (if we're in a tracking context)
	value := c.signal.value

	if !c.untracked && currentObserver != nil {
		switch observer := currentObserver.(type) {
		case *Computed[T]:
			// Add a subscription to this computed
			unsubscribe := c.signal.Subscribe(func(newValue T) {
				if observer.isStale {
					return
				}
				observer.isStale = true

				// If we're not within a batch, recompute immediately
				if !isBatching() {
					prevObserver := currentObserver
					currentObserver = observer
					observer.signal.Set(observer.compute())
					currentObserver = prevObserver
					observer.isStale = false
				}
			})

			// Store the unsubscribe function to clean up later
			if observer.cleanup == nil {
				observer.cleanup = unsubscribe
			} else {
				oldCleanup := observer.cleanup
				observer.cleanup = func() {
					oldCleanup()
					unsubscribe()
				}
			}
		case *Effect:
			// Add a subscription to this computed
			unsubscribe := c.signal.Subscribe(func(newValue T) {
				if !observer.disposed {
					execute(observer)
				}
			})

			// Store the unsubscribe function to clean up later
			if observer.cleanup == nil {
				observer.cleanup = unsubscribe
			} else {
				oldCleanup := observer.cleanup
				observer.cleanup = func() {
					oldCleanup()
					unsubscribe()
				}
			}
		}
	}

	return value
}

// GetUntracked returns the computed value without tracking
func (c *Computed[T]) GetUntracked() T {
	prev := c.untracked
	c.untracked = true
	value := c.Get()
	c.untracked = prev
	return value
}

// Peek gets the computed value without creating a dependency
func (c *Computed[T]) Peek() T {
	return c.GetUntracked()
}

// CreateMemo creates a memoized value (optimized computed)
func CreateMemo[T any](compute func() T, equals func(T, T) bool) *Memo[T] {
	if equals == nil {
		// Create a type-specific equality function that uses reflect.DeepEqual
		equals = func(a, b T) bool {
			return reflect.DeepEqual(a, b)
		}
	}

	// First compute the initial value
	initialValue := compute()

	// Create a signal with the initial value
	signal := Signal[T]{
		value:      initialValue,
		listeners:  make([]func(T), 0),
		equalityFn: equals,
		context:    activeContext,
	}

	// Now create the computed with this pre-set signal
	computed := &Computed[T]{
		signal:  signal,
		compute: compute,
		deps:    make([]any, 0),
		context: activeContext,
		isStale: false, // Not stale since we just computed the value
	}

	// Create the memo with the computed
	memo := &Memo[T]{
		Computed: *computed,
		equals:   equals,
	}

	// Run a Get operation to establish dependencies
	prevObserver := currentObserver
	currentObserver = computed
	currentObserver = prevObserver

	// Register with context if needed
	if activeContext != nil {
		activeContext.disposables = append(activeContext.disposables, func() {
			if computed.cleanup != nil {
				computed.cleanup()
				computed.cleanup = nil
			}
		})
	}

	return memo
}

// Subscribe adds a listener to the memo value
func (m *Memo[T]) Subscribe(listener func(T)) func() {
	return m.Computed.signal.Subscribe(listener)
}

// CreateEffect creates a new effect that runs when its dependencies change
func CreateEffect(execute func()) *Effect {
	effect := &Effect{
		execute:  execute,
		deps:     make([]any, 0),
		disposed: false,
		context:  activeContext,
	}

	// Run the effect once to capture dependencies
	runningEffects = append(runningEffects, *effect)
	prevObserver := currentObserver
	currentObserver = effect
	execute()
	currentObserver = prevObserver
	runningEffects = runningEffects[:len(runningEffects)-1]

	if activeContext != nil {
		activeContext.disposables = append(activeContext.disposables, func() {
			effect.Dispose()
		})
	}

	return effect
}

func setupEffect(effect *Effect) {
	// Create a subscriber for each dependency
	for _, dep := range effect.deps {
		switch d := dep.(type) {
		case *Signal[any]:
			d.Subscribe(func(any) {
				if !effect.disposed {
					execute(effect)
				}
			})
		case *Computed[any]:
			d.signal.Subscribe(func(any) {
				if !effect.disposed {
					execute(effect)
				}
			})
		}
	}
}

func execute(effect *Effect) {
	if effect.disposed {
		return
	}

	// Run cleanup if needed
	if effect.cleanup != nil {
		effect.cleanup()
		effect.cleanup = nil
	}

	// Re-run the effect
	prevObserver := currentObserver
	currentObserver = effect
	effect.execute()
	currentObserver = prevObserver
}

// Dispose cleans up the effect
func (e *Effect) Dispose() {
	if e.disposed {
		return
	}

	e.disposed = true
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}
}

// OnCleanup registers a cleanup function for the current context
func OnCleanup(fn func()) {
	if activeContext != nil {
		activeContext.disposables = append(activeContext.disposables, fn)
	}
}

// OnMount registers a function to run when a component mounts
func OnMount(fn func()) {
	onMountHooks = append(onMountHooks, fn)
}

// RunOnMountHooks runs all registered onMount hooks
func RunOnMountHooks() {
	for _, hook := range onMountHooks {
		hook()
	}
	onMountHooks = []func(){}
}

// Untrack prevents dependency tracking within the provided function
func Untrack[T any](fn func() T) T {
	// Save the current observer
	prevObserver := currentObserver
	// Set current observer to nil to prevent tracking
	currentObserver = nil
	// Call the function with tracking disabled
	result := fn()
	// Restore the previous observer
	currentObserver = prevObserver
	return result
}

// Batch executes the function with batched updates
func Batch(fn func()) {
	batchMutex.Lock()
	wasBatching := isBatchingFlag
	isBatchingFlag = true
	batchMutex.Unlock()

	// Execute the batch function
	fn()

	// Only process updates at the outermost batch level
	if !wasBatching {
		batchMutex.Lock()
		// Capture the current batch queue
		updates := make([]func(), len(batchQueue))
		copy(updates, batchQueue)
		// Reset the batch state
		batchQueue = nil
		isBatchingFlag = false
		batchMutex.Unlock()

		// Execute all queued updates at once
		for _, update := range updates {
			update()
		}
	}
}

// CreateResource creates an async resource with loading/error states
func CreateResource[T any](fetcher func() (T, error)) *Resource[T] {
	initialState := ResourceState[T]{
		Loading: true,
		Error:   nil,
		Data:    *new(T),
	}

	resource := &Resource[T]{
		signal:   *NewSignal(initialState),
		fetcher:  fetcher,
		disposed: false,
	}

	// Create the refetch function
	resource.refetch = func() {
		if resource.disposed {
			return
		}

		// Set loading state
		resource.signal.Update(func(s ResourceState[T]) ResourceState[T] {
			s.Loading = true
			return s
		})

		// Execute the fetcher
		go func() {
			if resource.disposed {
				return
			}

			data, err := fetcher()

			// Update the resource with the result
			resource.signal.Update(func(s ResourceState[T]) ResourceState[T] {
				s.Loading = false
				s.Error = err
				if err == nil {
					s.Data = data
				}
				return s
			})
		}()
	}

	// Initial fetch
	resource.refetch()

	// Register with the current context
	if activeContext != nil {
		activeContext.disposables = append(activeContext.disposables, func() {
			resource.disposed = true
		})
	}

	return resource
}

// Read returns the current state of the resource
func (r *Resource[T]) Read() ResourceState[T] {
	return r.signal.Get()
}

// Loading returns whether the resource is currently loading
func (r *Resource[T]) Loading() bool {
	return r.signal.Get().Loading
}

// Error returns any error that occurred during fetching
func (r *Resource[T]) Error() error {
	return r.signal.Get().Error
}

// Data returns the data from the resource
func (r *Resource[T]) Data() T {
	return r.signal.Get().Data
}

// Refetch triggers the resource to fetch again
func (r *Resource[T]) Refetch() {
	r.refetch()
}

// CreateStore creates a reactive store similar to Solid's createStore
func CreateStore[T any](initialState T) *Store[T] {
	return &Store[T]{
		state:      *NewSignal(initialState),
		middleware: make([]func(path []string, value any, oldValue any) any, 0),
	}
}

// Get returns the current state of the store
func (s *Store[T]) Get() T {
	return s.state.Get()
}

// Set updates the entire store state
func (s *Store[T]) Set(newState T) {
	s.state.Set(newState)
}

// SetPath updates a nested path in the store
func (s *Store[T]) SetPath(path []string, value any) {
	s.state.Update(func(state T) T {
		return updatePath(state, path, value, s.middleware).(T)
	})
}

// Use adds middleware to the store
func (s *Store[T]) Use(middleware func(path []string, value any, oldValue any) any) {
	s.middleware = append(s.middleware, middleware)
}

// updatePath recursively updates a nested path in an object
func updatePath(obj any, path []string, value any, middleware []func(path []string, value any, oldValue any) any) any {
	if len(path) == 0 {
		// Apply middleware
		oldValue := obj
		newValue := value
		for _, mw := range middleware {
			newValue = mw(path, value, oldValue)
		}
		return newValue
	}

	if obj == nil {
		return nil
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Handle different container types
	switch v.Kind() {
	case reflect.Map:
		key := reflect.ValueOf(path[0])
		newMap := reflect.MakeMap(v.Type())

		// Copy all existing entries
		iter := v.MapRange()
		for iter.Next() {
			if iter.Key().String() == path[0] {
				// Update this key with the recursive result
				newValue := updatePath(iter.Value().Interface(), path[1:], value, middleware)
				newMap.SetMapIndex(iter.Key(), reflect.ValueOf(newValue))
			} else {
				// Keep the same value
				newMap.SetMapIndex(iter.Key(), iter.Value())
			}
		}

		// If the key doesn't exist yet, add it
		if !v.MapIndex(key).IsValid() {
			newMap.SetMapIndex(key, reflect.ValueOf(
				updatePath(nil, path[1:], value, middleware),
			))
		}

		return newMap.Interface()

	case reflect.Struct:
		field := v.FieldByName(path[0])
		if !field.IsValid() {
			return obj // Field doesn't exist, return unchanged
		}

		// Create a new struct with the updated field
		newStruct := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).Name == path[0] {
				fieldValue := updatePath(field.Interface(), path[1:], value, middleware)
				newStruct.Field(i).Set(reflect.ValueOf(fieldValue))
			} else {
				newStruct.Field(i).Set(v.Field(i))
			}
		}

		return newStruct.Interface()

	case reflect.Slice, reflect.Array:
		index := 0
		fmt.Sscanf(path[0], "%d", &index)

		if index < 0 || index >= v.Len() {
			return obj // Index out of bounds, return unchanged
		}

		// Create a new slice with the updated element
		newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		reflect.Copy(newSlice, v)

		// Update the element
		newElement := updatePath(v.Index(index).Interface(), path[1:], value, middleware)
		newSlice.Index(index).Set(reflect.ValueOf(newElement))

		return newSlice.Interface()

	default:
		return obj // Can't update, return unchanged
	}
}

func isBatching() bool {
	batchMutex.Lock()
	defer batchMutex.Unlock()
	return isBatchingFlag
}

func batchUpdate(update func()) {
	batchMutex.Lock()
	defer batchMutex.Unlock()
	batchQueue = append(batchQueue, update)
}

// Accessor is a helper to get/set a signal with a single variable
type Accessor[T any] struct {
	signal *Signal[T]
}

// NewAccessor creates a new accessor for a signal
func NewAccessor[T any](initialValue T) *Accessor[T] {
	return &Accessor[T]{
		signal: NewSignal(initialValue),
	}
}

// Call gets or sets the signal value
func (a *Accessor[T]) Call(args ...T) any {
	if len(args) == 0 {
		// Getter mode
		return a.signal.Get()
	}
	// Setter mode
	a.signal.Set(args[0])
	return nil
}

// Selector creates a derived signal that selects part of another signal
func Selector[T any, R any](source *Signal[T], selector func(T) R) *Computed[R] {
	// Create a computed that will track the source signal
	comp := NewComputed(func() R {
		// Explicitly read the source value to create a dependency
		sourceValue := source.Get()
		return selector(sourceValue)
	})

	// Ensure initial computation
	comp.isStale = true
	comp.Get()

	return comp
}

// DerivedSignal creates a two-way bindable signal derived from another
type DerivedSignal[T any, R any] struct {
	source *Signal[T]
	getter func(T) R
	setter func(T, R) T
}

// NewDerivedSignal creates a derived signal with custom get/set functions
func NewDerivedSignal[T any, R any](
	source *Signal[T],
	getter func(T) R,
	setter func(T, R) T,
) *DerivedSignal[T, R] {
	return &DerivedSignal[T, R]{
		source: source,
		getter: getter,
		setter: setter,
	}
}

// Get retrieves the derived value
func (d *DerivedSignal[T, R]) Get() R {
	return d.getter(d.source.Get())
}

// Set updates the source through the derived signal
func (d *DerivedSignal[T, R]) Set(value R) {
	current := d.source.Get()
	updated := d.setter(current, value)
	d.source.Set(updated)
}
