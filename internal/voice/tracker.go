// Package voice tracks voice sessions independently for each guild and user.
package voice

import (
	"sync"
	"time"
)

type Key struct{ GuildID, UserID string }
type State struct {
	ChannelID string
	Joined    time.Time
}
type Change struct {
	From, To      string
	Duration      time.Duration
	KnownDuration bool
}

type Tracker struct {
	mu     sync.Mutex
	states map[Key]State
}

func New() *Tracker { return &Tracker{states: make(map[Key]State)} }

// Seed resets a guild after a fresh Gateway session; existing occupants have unknown join times.
func (t *Tracker) Seed(guildID string, occupants map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key := range t.states {
		if key.GuildID == guildID {
			delete(t.states, key)
		}
	}
	for user, channel := range occupants {
		if channel != "" {
			t.states[Key{guildID, user}] = State{ChannelID: channel}
		}
	}
}

func (t *Tracker) Update(key Key, channelID, beforeChannelID string, now time.Time) (Change, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	previous, exists := t.states[key]
	if !exists {
		previous.ChannelID = beforeChannelID
	}
	if previous.ChannelID == channelID {
		return Change{}, false
	}
	change := Change{From: previous.ChannelID, To: channelID}
	if !previous.Joined.IsZero() && !now.Before(previous.Joined) {
		change.KnownDuration = true
		change.Duration = now.Sub(previous.Joined).Truncate(time.Second)
	}
	if channelID == "" {
		delete(t.states, key)
	} else {
		t.states[key] = State{ChannelID: channelID, Joined: now}
	}
	return change, true
}
