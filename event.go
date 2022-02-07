package ship

import (
	"fmt"
	"reflect"
	"sync"
)

// Event is a payload for a message.
type Event interface {
	// EventName returns the name of the event.
	EventName() string
}

var (
	// eventStore is global to hold event registration data.
	eventStore = make(map[string]reflect.Type)

	// eventStoreMu is a mutex for locking the event store.
	eventStoreMu = sync.RWMutex{}
)

// getType returns the reflected type of given value.
func getType(v interface{}) reflect.Type {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// RegisterEvent registers an event to be global available for serialization, and
// other dependents. It used to create concrete event data structs when loading
// from event store.
func RegisterEvent(e Event) {
	name := e.EventName()

	eventStoreMu.Lock()
	defer eventStoreMu.Unlock()

	if _, ok := eventStore[name]; ok {
		panic(fmt.Sprintf("ship: event %s is already registered", name))
	}
	eventStore[name] = getType(e)
}

// GetEvent returns a new instance of event matching it's name or an error if
// the event is not registered.
func GetEvent(name string) (Event, error) {
	eventStoreMu.RLock()
	defer eventStoreMu.RUnlock()

	if eventType, ok := eventStore[name]; ok {
		return reflect.New(eventType).Interface().(Event), nil
	}

	return nil, fmt.Errorf("ship: event %s is not registered", name)
}

// UnregisterEvent removes the event from registered events list.
// This is mainly useful in mainenance situations where the event data
// needs to be switched in a migrations or test.
func UnregisterEvent(event Event) {
	name := event.EventName()
	if _, ok := eventStore[name]; !ok {
		panic(fmt.Sprintf("ship: event %s is not registered", name))
	}

	delete(eventStore, name)
}
