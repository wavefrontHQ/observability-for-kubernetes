package broadcaster

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Broadcaster[M any] struct {
	mu            sync.Mutex
	subscriptions map[int]chan<- M
	nextID        int
}

func New[M any]() *Broadcaster[M] {
	return &Broadcaster[M]{
		mu:            sync.Mutex{},
		subscriptions: map[int]chan<- M{},
	}
}

func (s *Broadcaster[M]) Subscribe() (<-chan M, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	messages := make(chan M)
	s.subscriptions[id] = messages
	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.subscriptions, id)
		close(messages)
	}
	log.WithFields(log.Fields{
		"id":        id,
		"component": "broadcaster.Broadcaster",
	}).Debugf("%d subscribed", id)
	return messages, unsubscribe
}

func (s *Broadcaster[M]) Publish(timeout time.Duration, message M) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wg := &sync.WaitGroup{}
	wg.Add(len(s.subscriptions))
	for id, messages := range s.subscriptions {
		go func(id int, messages chan<- M) {
			defer wg.Done()
			t := time.NewTimer(timeout)
			defer t.Stop()
			select {
			case messages <- message:
			case <-t.C:
				log.WithFields(log.Fields{
					"id":        id,
					"component": "broadcaster.Broadcaster",
				}).Warnf("timeout publishing to %d", id)
			}
		}(id, messages)
	}
	wg.Wait()
}
