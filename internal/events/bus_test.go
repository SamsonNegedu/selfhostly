package events

import "testing"

func TestPublishReachesEverySubscriber(t *testing.T) {
	bus := NewBus()
	a, stopA := bus.Subscribe()
	b, stopB := bus.Subscribe()
	defer stopA()
	defer stopB()

	bus.Publish(Event{Kind: KindApps, ID: "a1"})

	for name, ch := range map[string]<-chan Event{"a": a, "b": b} {
		select {
		case got := <-ch:
			if got.Kind != KindApps || got.ID != "a1" {
				t.Errorf("%s got %+v", name, got)
			}
		default:
			t.Errorf("%s got nothing", name)
		}
	}
}

func TestSlowSubscriberNeverBlocksPublisher(t *testing.T) {
	bus := NewBus()
	_, stop := bus.Subscribe()
	defer stop()

	for i := 0; i < subscriberBuffer*3; i++ {
		bus.Publish(Event{Kind: KindJobs, ID: "j"})
	}
}

func TestStoppedSubscriberGetsNothing(t *testing.T) {
	bus := NewBus()
	ch, stop := bus.Subscribe()
	stop()
	bus.Publish(Event{Kind: KindNodes, ID: "n"})
	select {
	case <-ch:
		t.Fatal("a stopped subscriber must not receive events")
	default:
	}
}
