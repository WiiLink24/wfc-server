package logging

import "sync"

type eventCallbackConfig struct {
	Function   func(eventType string, eventData map[string]any)
	EventTypes map[string]struct{}
	AllEvents  bool
}

var (
	eventCallbacks []eventCallbackConfig
	mutex          sync.RWMutex
)

func Event(eventType string, eventData map[string]any) {
	mutex.RLock()
	defer mutex.RUnlock()
	for _, callback := range eventCallbacks {
		if callback.AllEvents {
			go callback.Function(eventType, eventData)
		} else if _, ok := callback.EventTypes[eventType]; ok {
			go callback.Function(eventType, eventData)
		}
	}
}

func RegisterEventCallback(eventTypes []string, callback func(eventType string, eventData map[string]any)) {
	eventTypeSet := make(map[string]struct{})
	allEvents := false
	for _, eventType := range eventTypes {
		if eventType == "all" {
			allEvents = true
			eventTypeSet = make(map[string]struct{})
			break
		}
		eventTypeSet[eventType] = struct{}{}
	}

	mutex.Lock()
	defer mutex.Unlock()
	eventCallbacks = append(eventCallbacks, eventCallbackConfig{
		Function:   callback,
		EventTypes: eventTypeSet,
		AllEvents:  allEvents,
	})
}
