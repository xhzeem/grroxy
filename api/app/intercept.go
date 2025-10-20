package api

import (
	"log"
	"sync"

	"github.com/pocketbase/pocketbase/core"
)

// InterceptUpdateChannels stores channels for each intercept waiting goroutine
var (
	interceptChannels   = make(map[string]chan InterceptUpdate)
	interceptChannelsMu sync.RWMutex
)

// RegisterInterceptChannel registers a channel for a specific intercept ID
func RegisterInterceptChannel(id string, ch chan InterceptUpdate) {
	interceptChannelsMu.Lock()
	defer interceptChannelsMu.Unlock()
	interceptChannels[id] = ch
}

// UnregisterInterceptChannel removes the channel for a specific intercept ID
func UnregisterInterceptChannel(id string) {
	interceptChannelsMu.Lock()
	defer interceptChannelsMu.Unlock()
	if ch, exists := interceptChannels[id]; exists {
		close(ch)
		delete(interceptChannels, id)
	}
}

// NotifyInterceptUpdate sends an update to the waiting goroutine
func NotifyInterceptUpdate(id string, update InterceptUpdate) {
	interceptChannelsMu.RLock()
	defer interceptChannelsMu.RUnlock()
	if ch, exists := interceptChannels[id]; exists {
		select {
		case ch <- update:
			log.Printf("[InterceptManager] Notified waiting goroutine for ID=%s", id)
		default:
			log.Printf("[InterceptManager][WARN] Channel for ID=%s is not ready", id)
		}
	}
}

// SetupInterceptHooks sets up all the event hooks for intercept management
// This replaces SDK-based realtime subscriptions with native PocketBase hooks
func (backend *Backend) SetupInterceptHooks() error {
	log.Println("[InterceptManager] Setting up intercept hooks...")

	// Hook 1: Monitor intercept state changes in _settings
	backend.App.OnRecordAfterUpdateRequest("_settings").Add(func(e *core.RecordUpdateEvent) error {
		// Check if this is the INTERCEPT setting (ID has 6 underscores, not 8)
		if e.Record.Id != "INTERCEPT______" {
			return nil
		}

		value := e.Record.GetString("value")
		log.Printf("[InterceptManager] Intercept setting changed to: %s", value)

		if value == "false" {
			// Intercept turned OFF - forward all pending intercepts
			log.Println("[InterceptManager] Intercept disabled - forwarding all pending requests")

			if PROXY != nil {
				PROXY.Intercept = false
			}

			// Forward all pending intercepts
			go backend.forwardAllIntercepts()
		} else {
			// Intercept turned ON
			log.Println("[InterceptManager] Intercept enabled")

			if PROXY != nil {
				PROXY.Intercept = true
			}
		}

		return nil
	})

	// Hook 2: Handle new intercept requests being created
	backend.App.OnRecordAfterCreateRequest("_intercept").Add(func(e *core.RecordCreateEvent) error {
		log.Printf("[InterceptManager] New intercept request created: ID=%s", e.Record.Id)
		// Frontend will display this via realtime subscription
		// No action needed on backend side
		return nil
	})

	// Hook 3: Handle intercept updates (forward/drop actions)
	backend.App.OnRecordAfterUpdateRequest("_intercept").Add(func(e *core.RecordUpdateEvent) error {
		action := e.Record.GetString("action")
		log.Printf("[InterceptManager] Intercept updated: ID=%s Action=%s", e.Record.Id, action)

		if action == "forward" || action == "drop" {
			log.Printf("[InterceptManager] Intercept action received: %s for ID=%s", action, e.Record.Id)

			// Notify the waiting goroutine via channel
			update := InterceptUpdate{
				Action:       action,
				IsReqEdited:  e.Record.GetBool("is_req_edited"),
				IsRespEdited: e.Record.GetBool("is_resp_edited"),
			}
			NotifyInterceptUpdate(e.Record.Id, update)
		}

		return nil
	})

	log.Println("[InterceptManager] Intercept hooks registered successfully")
	return nil
}

// forwardAllIntercepts forwards all pending intercept requests when intercept is disabled
func (backend *Backend) forwardAllIntercepts() {
	interceptChannelsMu.RLock()
	defer interceptChannelsMu.RUnlock()

	if len(interceptChannels) == 0 {
		log.Println("[InterceptManager] No pending intercepts to forward")
		return
	}

	log.Printf("[InterceptManager] Forwarding %d pending intercepts via channels", len(interceptChannels))

	// Directly notify all waiting goroutines via their channels
	// Each goroutine will handle deleting its own record
	forwardUpdate := InterceptUpdate{
		Action:       "forward",
		IsReqEdited:  false,
		IsRespEdited: false,
	}

	for id, ch := range interceptChannels {
		select {
		case ch <- forwardUpdate:
			log.Printf("[InterceptManager] Forwarded intercept %s via channel", id)
		default:
			log.Printf("[InterceptManager][WARN] Channel for ID=%s is not ready", id)
		}
	}

	log.Println("[InterceptManager] All pending intercepts forwarded via channels")
}

// UpdateInterceptFilters updates the intercept filters for the proxy
func (backend *Backend) UpdateInterceptFilters(filters string) {
	if PROXY != nil {
		PROXY.Filters = filters
		log.Printf("[InterceptManager] Updated intercept filters: %s", filters)
	}
}
