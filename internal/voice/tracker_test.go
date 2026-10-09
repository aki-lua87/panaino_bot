package voice

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSessionsAreIsolatedAcrossGuilds(t *testing.T) {
	tracker := New()
	now := time.Now()
	a, b := Key{"guild-a", "same-user"}, Key{"guild-b", "same-user"}
	tracker.Update(a, "voice-a", "", now)
	tracker.Update(b, "voice-b", "", now.Add(time.Minute))
	change, changed := tracker.Update(a, "", "voice-a", now.Add(2*time.Minute+500*time.Millisecond))
	if !changed || !change.KnownDuration || change.Duration != 2*time.Minute {
		t.Fatalf("wrong guild A duration: %+v", change)
	}
	change, changed = tracker.Update(b, "", "voice-b", now.Add(3*time.Minute))
	if !changed || !change.KnownDuration || change.Duration != 2*time.Minute {
		t.Fatalf("wrong guild B duration: %+v", change)
	}
}

func TestMoveMuteAndLeave(t *testing.T) {
	tracker := New()
	now := time.Now()
	key := Key{"guild", "user"}
	tracker.Update(key, "first", "", now)
	if _, changed := tracker.Update(key, "first", "first", now.Add(time.Minute)); changed {
		t.Fatal("mute event produced a notification")
	}
	change, changed := tracker.Update(key, "second", "first", now.Add(2*time.Minute))
	if !changed || change.From != "first" || change.To != "second" || change.Duration != 2*time.Minute {
		t.Fatalf("move: %+v", change)
	}
	change, _ = tracker.Update(key, "", "second", now.Add(3*time.Minute))
	if change.Duration != time.Minute {
		t.Fatalf("move didn't reset channel duration: %+v", change)
	}
	if _, changed := tracker.Update(key, "", "", now); changed {
		t.Fatal("duplicate leave produced a notification")
	}
}

func TestReconnectDoesNotInventJoinTime(t *testing.T) {
	tracker := New()
	key := Key{"guild", "user"}
	tracker.Update(key, "vc", "", time.Now().Add(-time.Hour))
	tracker.Seed("guild", map[string]string{"user": "vc"})
	change, changed := tracker.Update(key, "", "vc", time.Now())
	if !changed || change.KnownDuration {
		t.Fatalf("reconnect duration must be unknown: %+v", change)
	}
	change, changed = tracker.Update(Key{"other", "user"}, "", "vc", time.Now())
	if !changed || change.KnownDuration {
		t.Fatalf("untracked leave must not invent duration: %+v", change)
	}
}

func TestConcurrentUpdates(t *testing.T) {
	tracker := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := Key{fmt.Sprint(i % 2), fmt.Sprint(i)}
			tracker.Update(key, "vc", "", time.Now())
			tracker.Update(key, "", "vc", time.Now())
		}(i)
	}
	wg.Wait()
}
