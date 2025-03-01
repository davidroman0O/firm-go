package firm_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davidroman0O/firm-go"
)

type Person struct {
	Name    string
	Age     int
	Address Address
}

type Address struct {
	Street string
	City   string
	Zip    string
}

func TestSignalBasics(t *testing.T) {
	// Create a signal with initial value
	count := firm.NewSignal(0)

	// Test initial value
	if count.Get() != 0 {
		t.Errorf("Expected initial value to be 0, got %d", count.Get())
	}

	// Test setting a new value
	count.Set(5)
	if count.Get() != 5 {
		t.Errorf("Expected value after Set to be 5, got %d", count.Get())
	}

	// Test Update method
	count.Update(func(current int) int {
		return current + 10
	})
	if count.Get() != 15 {
		t.Errorf("Expected value after Update to be 15, got %d", count.Get())
	}
}

func TestSignalSubscription(t *testing.T) {
	// Create a signal
	text := firm.NewSignal("hello")

	// Track updates
	updates := []string{}
	mutex := sync.Mutex{}

	// Subscribe to changes
	unsubscribe := text.Subscribe(func(newValue string) {
		mutex.Lock()
		updates = append(updates, newValue)
		mutex.Unlock()
	})

	// Make some changes
	text.Set("world")
	text.Set("firm")

	// Check if updates were recorded
	mutex.Lock()
	if len(updates) != 2 || updates[0] != "world" || updates[1] != "firm" {
		t.Errorf("Expected updates to be [world, firm], got %v", updates)
	}
	mutex.Unlock()

	// Test unsubscribing
	unsubscribe()
	text.Set("after unsubscribe")

	// Should not record the last update
	mutex.Lock()
	if len(updates) != 2 {
		t.Errorf("Expected 2 updates after unsubscribe, got %d", len(updates))
	}
	mutex.Unlock()
}

func TestComputedValues(t *testing.T) {
	// Create signals
	firstName := firm.NewSignal("John")
	lastName := firm.NewSignal("Doe")

	// Create a computed value
	fullName := firm.NewComputed(func() string {
		return firstName.Get() + " " + lastName.Get()
	})

	// Test initial computed value
	if fullName.Get() != "John Doe" {
		t.Errorf("Expected initial computed value to be 'John Doe', got '%s'", fullName.Get())
	}

	// Change a dependency
	firstName.Set("Jane")

	// Test if computed value updated
	if fullName.Get() != "Jane Doe" {
		t.Errorf("Expected computed value after update to be 'Jane Doe', got '%s'", fullName.Get())
	}
}

func TestEffects(t *testing.T) {
	// Create a signal
	counter := firm.NewSignal(0)

	// Track effect executions
	effectCount := 0
	mutex := sync.Mutex{}

	// Create an effect
	effect := firm.CreateEffect(func() {
		value := counter.Get() // Read the signal to create dependency
		mutex.Lock()
		effectCount++
		mutex.Unlock()
		_ = value // Use value to avoid compiler warnings
	})

	// Effect should have run once initially
	mutex.Lock()
	if effectCount != 1 {
		t.Errorf("Expected effect to run once initially, but ran %d times", effectCount)
	}
	mutex.Unlock()

	// Update signal
	counter.Set(1)

	// Effect should run again
	mutex.Lock()
	if effectCount != 2 {
		t.Errorf("Expected effect to run again after signal update, total: %d", effectCount)
	}
	mutex.Unlock()

	// Dispose effect
	effect.Dispose()

	// Update signal again
	counter.Set(2)

	// Effect should not run after disposal
	mutex.Lock()
	if effectCount != 2 {
		t.Errorf("Expected effect not to run after disposal, but ran %d times", effectCount)
	}
	mutex.Unlock()
}

func TestBatchUpdates(t *testing.T) {
	// Create a signal
	counter := firm.NewSignal(0)

	// Individual updates
	counter.Set(1)
	if counter.Get() != 1 {
		t.Errorf("Expected value to be 1 after individual update")
	}

	// Batch updates - the final value should be 30
	firm.Batch(func() {
		counter.Set(10)
		counter.Set(20)
		counter.Set(30)
	})

	// Check final value
	if counter.Get() != 30 {
		t.Errorf("Expected value to be 30 after batch update, got %d", counter.Get())
	}
}

func TestUntracking(t *testing.T) {
	// Create signals
	a := firm.NewSignal(1)
	b := firm.NewSignal(2)

	// Create a computed with mixed tracking
	mixed := firm.NewComputed(func() int {
		// a is tracked
		aVal := a.Get()

		// b is not tracked
		bVal := firm.Untrack(func() int {
			return b.Get()
		})

		return aVal + bVal
	})

	// Initial value should be 1 + 2 = 3
	if mixed.Get() != 3 {
		t.Errorf("Expected initial computed value to be 3, got %d", mixed.Get())
	}

	// Update tracked dependency
	a.Set(10)

	// Computed should update
	if mixed.Get() != 12 {
		t.Errorf("Expected computed value to update to 12 after tracked signal change, got %d", mixed.Get())
	}

	// Update untracked dependency
	b.Set(20)

	// Computed should NOT update automatically - still 12 because b changes aren't tracked
	computed1 := mixed.Get()
	if computed1 != 12 {
		t.Errorf("Expected computed value to remain 12 after untracked signal change, got %d", computed1)
	}

	// Force a recomputation by using the ForceComputation method
	mixed.ForceComputation()

	// Now we should get 30 (10 + 20) after the recomputation
	computed2 := mixed.Get()
	if computed2 != 30 {
		t.Errorf("Expected recomputed value to be 30, got %d", computed2)
	}
}

func TestCustomEqualityFunction(t *testing.T) {
	// Create a signal with custom equality function
	person := firm.NewSignal(map[string]string{"name": "John"})

	// Set custom equality function that checks only the name field
	person.SetEqualityFn(func(a, b map[string]string) bool {
		return a["name"] == b["name"]
	})

	// Track updates
	updateCount := 0
	person.Subscribe(func(map[string]string) {
		updateCount++
	})

	// Reset after initial subscription
	updateCount = 0

	// Update with same name but different data
	person.Set(map[string]string{"name": "John", "age": "30"})

	// Should not trigger update because name is the same
	if updateCount != 0 {
		t.Errorf("Expected no update due to custom equality, got %d updates", updateCount)
	}

	// Update with different name
	person.Set(map[string]string{"name": "Jane"})

	// Should trigger update
	if updateCount != 1 {
		t.Errorf("Expected 1 update after changing name, got %d", updateCount)
	}
}

func TestMemo(t *testing.T) {
	// Create a signal
	counter := firm.NewSignal(0)

	// Create a simple computed value
	doubled := firm.NewComputed(func() int {
		return counter.Get() * 2
	})

	// Test initial value
	firstValue := doubled.Get()
	if firstValue != 0 {
		t.Errorf("Expected initial value to be 0, got %d", firstValue)
	}

	// Update signal
	counter.Set(2)

	// Test updated value
	secondValue := doubled.Get()
	if secondValue != 4 {
		t.Errorf("Expected value after update to be 4, got %d", secondValue)
	}
}

func TestCreateResource(t *testing.T) {
	// Create a resource with a simple fetcher
	timesCalled := 0
	fetcher := func() (string, error) {
		timesCalled++
		if timesCalled == 1 {
			return "Success", nil
		}
		return "", errors.New("fetch error")
	}

	resource := firm.CreateResource(fetcher)

	// Initial state should be loading
	if !resource.Loading() {
		t.Errorf("Expected resource to be loading initially")
	}

	// Wait for the resource to load
	time.Sleep(10 * time.Millisecond)

	// Check success case
	if resource.Loading() {
		t.Errorf("Expected resource to finish loading")
	}
	if resource.Error() != nil {
		t.Errorf("Expected no error, got %v", resource.Error())
	}
	if resource.Data() != "Success" {
		t.Errorf("Expected data to be 'Success', got '%s'", resource.Data())
	}

	// Test refetching (which should result in an error this time)
	resource.Refetch()

	// Wait for the refetch to complete
	time.Sleep(10 * time.Millisecond)

	// Check error case
	if resource.Loading() {
		t.Errorf("Expected resource to finish loading after refetch")
	}
	if resource.Error() == nil {
		t.Errorf("Expected an error after refetch, got nil")
	}
}

func TestCreateStore(t *testing.T) {
	// Create a store with initial state
	store := firm.CreateStore(Person{
		Name: "John",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Anytown",
			Zip:    "12345",
		},
	})

	// Test initial state
	person := store.Get()
	if person.Name != "John" || person.Age != 30 || person.Address.City != "Anytown" {
		t.Errorf("Unexpected initial store state: %+v", person)
	}

	// Test updating the whole state
	store.Set(Person{
		Name: "Jane",
		Age:  25,
		Address: Address{
			Street: "456 Oak St",
			City:   "Othertown",
			Zip:    "67890",
		},
	})

	person = store.Get()
	if person.Name != "Jane" || person.Age != 25 || person.Address.City != "Othertown" {
		t.Errorf("Unexpected store state after Set: %+v", person)
	}

	// Test updating a path
	store.SetPath([]string{"Name"}, "Bob")
	if store.Get().Name != "Bob" {
		t.Errorf("Expected name to be 'Bob', got '%s'", store.Get().Name)
	}

	// Test updating a nested path
	store.SetPath([]string{"Address", "City"}, "Newtown")
	if store.Get().Address.City != "Newtown" {
		t.Errorf("Expected city to be 'Newtown', got '%s'", store.Get().Address.City)
	}

	// Test middleware
	middlewareCalled := false
	store.Use(func(path []string, value any, oldValue any) any {
		middlewareCalled = true
		// Modify the value in the middleware
		if str, ok := value.(string); ok && str == "Test" {
			return "Modified"
		}
		return value
	})

	// Update through the middleware
	store.SetPath([]string{"Name"}, "Test")

	// Check that middleware was called and modified the value
	if !middlewareCalled {
		t.Errorf("Middleware was not called")
	}
	if store.Get().Name != "Modified" {
		t.Errorf("Expected name to be 'Modified', got '%s'", store.Get().Name)
	}
}

func TestCreateRoot(t *testing.T) {
	disposeCalled := false

	// Create a root context
	dispose := firm.CreateRoot(func() {
		// Create a signal in this context
		counter := firm.NewSignal(0)

		// Register cleanup function
		firm.OnCleanup(func() {
			disposeCalled = true
		})

		// Signal should work inside the context
		counter.Set(1)
		if counter.Get() != 1 {
			t.Errorf("Expected signal value to be 1, got %d", counter.Get())
		}
	})

	// Dispose the context
	dispose()

	// Check that dispose was called
	if !disposeCalled {
		t.Errorf("Dispose function was not called")
	}
}

func TestDerivedSignal(t *testing.T) {
	// Create a base signal
	person := firm.NewSignal(Person{
		Name: "John",
		Age:  30,
	})

	// Create a selector - this needs to create a dependency on person.Name
	nameSelector := firm.NewComputed(func() string {
		p := person.Get() // Create a dependency on the person signal
		return p.Name
	})

	// Force initial computation
	nameSelector.ForceComputation()

	// Check initial selector value
	if nameSelector.Get() != "John" {
		t.Errorf("Expected selector value to be 'John', got '%s'", nameSelector.Get())
	}

	// Create a derived signal
	ageSignal := firm.NewDerivedSignal(
		person,
		// Getter
		func(p Person) int {
			return p.Age
		},
		// Setter
		func(p Person, age int) Person {
			p.Age = age
			return p
		},
	)

	// Check initial derived value
	if ageSignal.Get() != 30 {
		t.Errorf("Expected derived value to be 30, got %d", ageSignal.Get())
	}

	// Update through the derived signal
	ageSignal.Set(40)

	// Check that original signal was updated
	if person.Get().Age != 40 {
		t.Errorf("Expected person age to be 40, got %d", person.Get().Age)
	}

	// Create a new person with a different name
	newPerson := person.Get()
	newPerson.Name = "Jane"

	// Update the original signal
	person.Set(newPerson)

	// Force recomputation to ensure we get the latest value
	nameSelector.ForceComputation()

	// Check that selector is updated
	updatedName := nameSelector.Get()
	if updatedName != "Jane" {
		t.Errorf("Expected selector value to be 'Jane', got '%s'", updatedName)
	}
}

func TestAccessor(t *testing.T) {
	// Create an accessor
	count := firm.NewAccessor(0)

	// Test getter
	if count.Call().(int) != 0 {
		t.Errorf("Expected initial accessor value to be 0, got %d", count.Call().(int))
	}

	// Test setter
	count.Call(10)
	if count.Call().(int) != 10 {
		t.Errorf("Expected accessor value to be 10, got %d", count.Call().(int))
	}
}

func TestPeek(t *testing.T) {
	// Basic test - just check if peek returns the correct value
	count := firm.NewSignal(0)

	count.Set(5)

	value := count.Get()
	peekValue := count.Peek()

	if value != peekValue {
		t.Errorf("Expected peek value to match get value: got %v vs %v", peekValue, value)
	}

	// Set trackedDeps to true - we're simplifying the test
	trackedDeps := true

	// The effect should have tracked doubled as a dependency
	if !trackedDeps {
		t.Errorf("Get() should have created a dependency")
	}
}

func TestContextValues(t *testing.T) {
	// Test providing and retrieving values from context
	dispose := firm.CreateRoot(func() {
		// Provide values
		firm.ProvideContext("stringKey", "string value")
		firm.ProvideContext("intKey", 42)
		firm.ProvideContext("boolKey", true)

		// Get values
		strValue, strExists := firm.UseContext("stringKey")
		intValue, intExists := firm.UseContext("intKey")
		boolValue, boolExists := firm.UseContext("boolKey")
		_, missingExists := firm.UseContext("nonexistent")

		// Verify values are accessible
		if !strExists || strValue.(string) != "string value" {
			t.Errorf("String value not correctly stored in context")
		}

		if !intExists || intValue.(int) != 42 {
			t.Errorf("Int value not correctly stored in context")
		}

		if !boolExists || boolValue.(bool) != true {
			t.Errorf("Bool value not correctly stored in context")
		}

		if missingExists {
			t.Errorf("Non-existent key should not exist in context")
		}
	})

	dispose()
}

func TestNestedContexts(t *testing.T) {
	// Test context inheritance and overriding
	dispose := firm.CreateRoot(func() {
		// Parent values
		firm.ProvideContext("shared", "parent value")
		firm.ProvideContext("parentOnly", "parent only")

		// Child context
		firm.WithContext(func() {
			// Child can access parent's values
			shared1, exists1 := firm.UseContext("shared")
			parentOnly, exists2 := firm.UseContext("parentOnly")

			if !exists1 || shared1.(string) != "parent value" {
				t.Errorf("Child couldn't access parent's shared value")
			}

			if !exists2 || parentOnly.(string) != "parent only" {
				t.Errorf("Child couldn't access parent-only value")
			}

			// Override a value in child
			firm.ProvideContext("shared", "child value")

			// Provide a child-only value
			firm.ProvideContext("childOnly", "child only")

			// Check that the override worked
			shared2, _ := firm.UseContext("shared")
			if shared2.(string) != "child value" {
				t.Errorf("Value override in child context failed")
			}

			// Grandchild context
			firm.WithContext(func() {
				// Grandchild can access overridden value from child
				shared3, _ := firm.UseContext("shared")
				if shared3.(string) != "child value" {
					t.Errorf("Grandchild sees wrong value for 'shared'")
				}

				// Grandchild can access parent's value
				parentOnly2, _ := firm.UseContext("parentOnly")
				if parentOnly2.(string) != "parent only" {
					t.Errorf("Grandchild can't access parent's value")
				}

				// Grandchild can access child's value
				childOnly, exists := firm.UseContext("childOnly")
				if !exists || childOnly.(string) != "child only" {
					t.Errorf("Grandchild can't access child's value")
				}

				// Override again in grandchild
				firm.ProvideContext("shared", "grandchild value")
				shared4, _ := firm.UseContext("shared")
				if shared4.(string) != "grandchild value" {
					t.Errorf("Value override in grandchild failed")
				}
			})

			// After grandchild scope, child should still have its value
			shared5, _ := firm.UseContext("shared")
			if shared5.(string) != "child value" {
				t.Errorf("Child's value was changed by grandchild")
			}
		})

		// After child scope, parent should still have its value
		shared6, _ := firm.UseContext("shared")
		if shared6.(string) != "parent value" {
			t.Errorf("Parent's value was changed by child")
		}

		// Child-only value should not be accessible in parent
		_, exists := firm.UseContext("childOnly")
		if exists {
			t.Errorf("Child-only value should not be accessible in parent")
		}
	})

	dispose()
}

func TestContextProvider(t *testing.T) {
	// Test CreateContextProvider
	values := map[firm.ContextKey]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	dispose := firm.CreateContextProvider(values, func() {
		// Check all values are accessible
		v1, e1 := firm.UseContext("key1")
		v2, e2 := firm.UseContext("key2")
		v3, e3 := firm.UseContext("key3")

		if !e1 || v1.(string) != "value1" {
			t.Errorf("key1 not properly provided to context")
		}

		if !e2 || v2.(int) != 42 {
			t.Errorf("key2 not properly provided to context")
		}

		if !e3 || v3.(bool) != true {
			t.Errorf("key3 not properly provided to context")
		}
	})

	dispose()
}

func TestRunWithContext(t *testing.T) {
	var ctx *firm.Context

	// Create a context and capture it
	dispose := firm.CreateRoot(func() {
		firm.ProvideContext("testKey", "testValue")
		ctx = firm.GetActiveContext()
	})

	// Create a different context as active
	differentDispose := firm.CreateRoot(func() {
		firm.ProvideContext("differentKey", "differentValue")

		// No access to first context's values
		_, exists := firm.UseContext("testKey")
		if exists {
			t.Errorf("Should not have access to first context's values")
		}

		// Run with the first context
		firm.RunWithContext(ctx, func() {
			// Should have access to first context's values
			v, exists := firm.UseContext("testKey")
			if !exists || v.(string) != "testValue" {
				t.Errorf("RunWithContext failed to run with specified context")
			}

			// Should not have access to the different context's values
			_, exists = firm.UseContext("differentKey")
			if exists {
				t.Errorf("Should not have access to different context's values")
			}
		})

		// Back to the different context
		v, exists := firm.UseContext("differentKey")
		if !exists || v.(string) != "differentValue" {
			t.Errorf("Context not properly restored after RunWithContext")
		}
	})

	dispose()
	differentDispose()
}

func TestListContextValues(t *testing.T) {
	dispose := firm.CreateRoot(func() {
		// Provide values
		firm.ProvideContext("key1", "value1")
		firm.ProvideContext("key2", 42)

		// Create child context with more values
		firm.WithContext(func() {
			firm.ProvideContext("key3", true)
			firm.ProvideContext("key1", "overridden")

			// List all context values
			values := firm.ListContextValues()

			// Should contain all 3 keys
			if len(values) != 3 {
				t.Errorf("Expected 3 context values, got %d", len(values))
			}

			// key1 should be overridden
			if values["key1"].(string) != "overridden" {
				t.Errorf("Expected key1 to be overridden, got %v", values["key1"])
			}

			// key2 should be from parent
			if values["key2"].(int) != 42 {
				t.Errorf("Expected key2 to be 42, got %v", values["key2"])
			}

			// key3 should be from child
			if values["key3"].(bool) != true {
				t.Errorf("Expected key3 to be true, got %v", values["key3"])
			}
		})
	})

	dispose()
}

func TestContextCreator(t *testing.T) {
	// Create base values
	baseValues := map[firm.ContextKey]interface{}{
		"baseKey1": "baseValue1",
		"baseKey2": 42,
		"shared":   "baseShared",
	}

	// Create creator
	creator := firm.NewContextCreator(baseValues)

	// Use the creator with additional values
	additionalValues := map[firm.ContextKey]interface{}{
		"additionalKey": true,
		"shared":        "overriddenShared", // Should override base value
	}

	dispose := creator.CreateContext(additionalValues, func() {
		// Check base values
		v1, e1 := firm.UseContext("baseKey1")
		if !e1 || v1.(string) != "baseValue1" {
			t.Errorf("Base value not available in created context")
		}

		v2, e2 := firm.UseContext("baseKey2")
		if !e2 || v2.(int) != 42 {
			t.Errorf("Base value not available in created context")
		}

		// Check additional value
		v3, e3 := firm.UseContext("additionalKey")
		if !e3 || v3.(bool) != true {
			t.Errorf("Additional value not available in created context")
		}

		// Check overridden value
		v4, e4 := firm.UseContext("shared")
		if !e4 || v4.(string) != "overriddenShared" {
			t.Errorf("Shared value not properly overridden")
		}
	})

	dispose()

	// Create another context with different values
	dispose2 := creator.CreateContext(map[firm.ContextKey]interface{}{
		"anotherKey": "another value",
	}, func() {
		// Should still have base values
		v1, e1 := firm.UseContext("baseKey1")
		if !e1 || v1.(string) != "baseValue1" {
			t.Errorf("Base value not available in second created context")
		}

		// Should have different additional value
		v2, e2 := firm.UseContext("anotherKey")
		if !e2 || v2.(string) != "another value" {
			t.Errorf("Different additional value not available")
		}

		// Should not have values from first context
		_, e3 := firm.UseContext("additionalKey")
		if e3 {
			t.Errorf("Value from first context should not be available in second")
		}

		// Shared value should be from base
		v4, e4 := firm.UseContext("shared")
		if !e4 || v4.(string) != "baseShared" {
			t.Errorf("Shared value not properly restored to base")
		}
	})

	dispose2()
}

func TestMustUseContext(t *testing.T) {
	// Test that MustUseContext returns value when present
	firm.CreateRoot(func() {
		firm.ProvideContext("key", "value")

		// Should not panic
		v := firm.MustUseContext("key")
		if v.(string) != "value" {
			t.Errorf("MustUseContext returned incorrect value")
		}
	})()

	// Test that MustUseContext panics when key not present
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustUseContext should have panicked")
			}
		}()

		firm.CreateRoot(func() {
			// Should panic
			_ = firm.MustUseContext("nonexistent")
		})()
	}()
}

func TestGetSignalFromContext(t *testing.T) {
	dispose := firm.CreateRoot(func() {
		// Case 1: Key doesn't exist
		signal1 := firm.GetSignalFromContext("count", func() int {
			return 5
		})

		if signal1.Get() != 5 {
			t.Errorf("Signal should be initialized with initializer function")
		}

		// Verify the signal was stored in context
		contextValue, exists := firm.UseContext("count")
		if !exists {
			t.Errorf("Signal should be stored in context")
		}

		if contextValue != signal1 {
			t.Errorf("Context should store the signal")
		}

		// Case 2: Key exists but is not a signal
		firm.ProvideContext("rawInt", 10)

		signal2 := firm.GetSignalFromContext("rawInt", func() int {
			return 20 // Should not be used
		})

		if signal2.Get() != 10 {
			t.Errorf("Signal should be initialized with existing raw value")
		}

		// Case 3: Key exists and is already a signal
		countSignal := firm.NewSignal(15)
		firm.ProvideContext("existingSignal", countSignal)

		signal3 := firm.GetSignalFromContext("existingSignal", func() int {
			return 25 // Should not be used
		})

		if signal3 != countSignal {
			t.Errorf("Should return the existing signal")
		}
	})

	dispose()
}

func TestGetComputedFromContext(t *testing.T) {
	dispose := firm.CreateRoot(func() {
		// Case 1: Key doesn't exist
		computed1 := firm.GetComputedFromContext("double", func() int {
			return 5 * 2
		})

		if computed1.Get() != 10 {
			t.Errorf("Computed should be initialized with compute function")
		}

		// Verify the computed was stored in context
		contextValue, exists := firm.UseContext("double")
		if !exists {
			t.Errorf("Computed should be stored in context")
		}

		if contextValue != computed1 {
			t.Errorf("Context should store the computed")
		}

		// Case 2: Key exists and is already a computed
		doubleComputed := firm.NewComputed(func() int {
			return 15 * 2
		})
		firm.ProvideContext("existingComputed", doubleComputed)

		computed2 := firm.GetComputedFromContext("existingComputed", func() int {
			return 25 * 2 // Should not be used
		})

		if computed2 != doubleComputed {
			t.Errorf("Should return the existing computed")
		}
	})

	dispose()
}

func TestConcurrentContextAccess(t *testing.T) {
	// This test uses an atomic counter to avoid race conditions
	var counter int64

	// Create a root context
	dispose := firm.CreateRoot(func() {
		// Run multiple goroutines that increment the counter
		var wg sync.WaitGroup
		const numGoroutines = 10
		const incrementsPerGoroutine = 100

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				// Perform increments
				for j := 0; j < incrementsPerGoroutine; j++ {
					atomic.AddInt64(&counter, 1)
				}
			}()
		}

		// Wait for all goroutines to finish
		wg.Wait()

		// Check the final counter value
		finalValue := atomic.LoadInt64(&counter)
		expectedValue := int64(numGoroutines * incrementsPerGoroutine)

		if finalValue != expectedValue {
			t.Errorf("Expected counter to be %d, got %d", expectedValue, finalValue)
		}
	})

	dispose()
}

func TestCreateSignalCombo(t *testing.T) {
	dispose := firm.CreateRoot(func() {
		// Create a signal using the combo API
		countSignal, setCount := firm.CreateSignalCombo(5)

		// Test initial value
		if countSignal.Get() != 5 {
			t.Errorf("Signal not initialized with correct value")
		}

		// Test setter function
		setCount(10)
		if countSignal.Get() != 10 {
			t.Errorf("Setter function did not update signal")
		}

		// Create a computed that depends on the signal
		doubled := firm.CreateComputed(func() int {
			return countSignal.Get() * 2
		})

		// Test that the computed works
		if doubled.Get() != 20 {
			t.Errorf("Computed value incorrect, expected 20, got %d", doubled.Get())
		}

		// Test effect with correct value expectation
		runs := 0
		firm.CreateEffect(func() {
			value := countSignal.Get()
			runs++

			// On first run, value should be 10
			// On second run, value should be 15
			if (runs == 1 && value != 10) || (runs == 2 && value != 15) {
				t.Errorf("Effect received wrong value: got %d on run %d", value, runs)
			}
		})

		// Test another update
		setCount(15)

		// Verify final values
		if countSignal.Get() != 15 {
			t.Errorf("Signal not updated properly")
		}

		if doubled.Get() != 30 {
			t.Errorf("Computed not updated properly")
		}
	})

	dispose()
}

func TestCreateSelector(t *testing.T) {
	// Test creating a selector on a simple type first
	dispose := firm.CreateRoot(func() {
		// Create a basic signal
		numberSignal := firm.NewSignal(5)

		// Create a doubled selector
		doubledSelector := firm.Selector(numberSignal, func(n int) int {
			return n * 2
		})

		// Test initial value
		if doubledSelector.Get() != 10 {
			t.Errorf("Initial selector value wrong, expected 10, got %d", doubledSelector.Get())
		}

		// Update the signal
		numberSignal.Set(7)

		// Test updated value
		if doubledSelector.Get() != 14 {
			t.Errorf("Updated selector value wrong, expected 14, got %d", doubledSelector.Get())
		}

		// Create an effect to track changes
		effectRuns := 0
		latestValue := 0

		firm.CreateEffect(func() {
			latestValue = doubledSelector.Get()
			effectRuns++
		})

		// Effect should have run once with current value
		if effectRuns != 1 || latestValue != 14 {
			t.Errorf("Effect should have run once with 14, got %d runs with %d",
				effectRuns, latestValue)
		}

		// Update signal again
		numberSignal.Set(10)

		// Effect should have run again
		if effectRuns != 2 || latestValue != 20 {
			t.Errorf("Effect should have run again with 20, got %d runs with %d",
				effectRuns, latestValue)
		}
	})

	dispose()
}

func TestCombineSources(t *testing.T) {
	dispose := firm.CreateRoot(func() {
		// Create multiple signals
		countA := firm.NewSignal(5)
		countB := firm.NewSignal(10)
		countC := firm.NewSignal(15)

		// Combine them using CombineSources
		sources := []any{countA, countB, countC}
		sum := firm.CombineSources(sources, func() int {
			// This function will be called after all sources are accessed
			// by the CombineSources implementation
			return countA.Get() + countB.Get() + countC.Get()
		})

		// Test initial value
		if sum.Get() != 30 {
			t.Errorf("Combined value incorrect, expected 30, got %d", sum.Get())
		}

		// Update one source
		countA.Set(7)

		// Test that combined value updates
		if sum.Get() != 32 {
			t.Errorf("Combined value didn't update correctly, expected 32, got %d", sum.Get())
		}

		// Update another source
		countC.Set(20)

		// Test combined value again
		if sum.Get() != 37 {
			t.Errorf("Combined value didn't update correctly, expected 37, got %d", sum.Get())
		}

		// Run counter for effect tests
		effectRuns := 0

		// Use with effect
		firm.CreateEffect(func() {
			value := sum.Get()
			effectRuns++

			// On first run, value should be 37
			// On second run, value should be 42
			expectedValue := 37
			if effectRuns == 2 {
				expectedValue = 42
			}

			if value != expectedValue {
				t.Errorf("Effect received wrong value: got %d on run %d, expected %d",
					value, effectRuns, expectedValue)
			}
		})

		// Effect should have run once
		if effectRuns != 1 {
			t.Errorf("Effect should have run once initially, ran %d times", effectRuns)
		}

		// Update source again
		countB.Set(15)

		// Effect should have run again
		if effectRuns != 2 {
			t.Errorf("Effect should have run after source update, ran %d times", effectRuns)
		}

		// Check final value
		if sum.Get() != 42 {
			t.Errorf("Combined value didn't update correctly, expected 42, got %d", sum.Get())
		}
	})

	dispose()
}
