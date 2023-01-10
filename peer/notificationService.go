package peer

import "time"

type NotificationService[K comparable, V any] interface {
	Wait(notificationKey K, timeout time.Duration) (V, error)
	Notify(notificationKey K, data V) bool
	ExecuteAndWait(waitingF func() error, notificationKey K, timeout time.Duration) (V, error)
}
