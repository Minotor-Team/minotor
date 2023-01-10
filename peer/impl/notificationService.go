package impl

import (
	"sync"
	"time"

	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/transport"
	"golang.org/x/xerrors"
)

func defaultValue[V any]() V {
	var res V
	return res
}

// A notification service to notify the completion of the route.
type NotificationServiceImpl[K comparable, V any] struct {
	lock     sync.RWMutex
	channels map[K]*chan V
}

func NewNotificationService[K comparable, V any]() peer.NotificationService[K, V] {
	return &NotificationServiceImpl[K, V]{
		channels: make(map[K]*chan V),
	}
}

// Wait for the notification of the route with the given id.
// Returns the tail of the route and true, if it succeeded within the timeout.
// Returns false otherwise.
// A timeout of 0 means that the function will wait indefinitely.
func (n *NotificationServiceImpl[K, V]) Wait(id K, timeout time.Duration) (V, error) {
	v, err := n.ExecuteAndWait(func() error { return nil }, id, timeout)
	if err != nil {
		return v, xerrors.Errorf("Wait: %v", err)
	}

	return v, nil
}

func (n *NotificationServiceImpl[K, V]) ExecuteAndWait(waitingF func() error, id K, timeout time.Duration) (V, error) {
	timer := time.NewTimer(timeout)
	ch := make(chan V, 1)
	n.lock.Lock()
	n.channels[id] = &ch
	n.lock.Unlock()
	errChan := make(chan error)
	go func(c chan error) {
		err := waitingF()
		if err != nil {
			c <- err
		}
	}(errChan)

	noTimeout := func(v V, ok bool) (V, error) {
		var err error = nil
		if !ok {
			err = xerrors.Errorf("ExecuteAndWait: the channel was closed and can't be notified")
		}
		return v, err
	}

	if timeout > 0 {
		select {
		case v, ok := <-ch:
			return noTimeout(v, ok)
		case <-timer.C:
			// We ensure that no one can notify for this id anymore.
			n.lock.Lock()
			delete(n.channels, id)
			n.lock.Unlock()
			return defaultValue[V](), transport.TimeoutError(timeout)
		case err := <-errChan:
			return defaultValue[V](), xerrors.Errorf("ExecuteAndWait: failed to execute the waiting function: %v", err)
		}
	} else {
		select {
		case v, ok := <-ch:
			return noTimeout(v, ok)
		case err := <-errChan:
			return defaultValue[V](), xerrors.Errorf("ExecuteAndWait: failed to execute the waiting function: %v", err)
		}
	}
}

func (n *NotificationServiceImpl[K, V]) Notify(id K, data V) bool {
	n.lock.Lock()
	defer n.lock.Unlock()
	ch, ok := n.channels[id]
	if !ok {
		return false
	}
	*ch <- data
	delete(n.channels, id)
	return true
}
