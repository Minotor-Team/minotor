package concurrent

import (
	"errors"
	"math/rand"
	"sync"

	"go.dedis.ch/cs438/datastructures"
)

// Thread safe slice. Any operation is guaranteed to be thread-safe.
type Slice[T any] struct {
	sync.RWMutex
	elements []T
}

func NewSlice[T any]() Slice[T] {
	return Slice[T]{
		elements: make([]T, 0),
	}
}

func (slice *Slice[T]) Append(elem T) {
	slice.Lock()
	defer slice.Unlock()
	slice.elements = append(slice.elements, elem)
}

func (slice *Slice[T]) AppendAll(elems []T) {
	slice.Lock()
	defer slice.Unlock()
	slice.elements = append(slice.elements, elems...)
}

func (slice *Slice[T]) Elements() []T {
	slice.Lock()
	defer slice.Unlock()
	res := make([]T, len(slice.elements))
	copy(res, slice.elements)
	return res
}

func (slice *Slice[T]) Get(i int) T {
	slice.RLock()
	defer slice.RUnlock()
	return slice.elements[i]
}

func (slice *Slice[T]) RandomElement() (T, error) {
	slice.RLock()
	defer slice.RUnlock()

	n := len(slice.elements)
	if n <= 0 {
		var res T
		return res, errors.New("empty slice")
	}

	idx := rand.Intn(n)
	return slice.elements[idx], nil
}

func (slice *Slice[T]) Len() int {
	slice.RLock()
	defer slice.RUnlock()
	return len(slice.elements)
}

func (slice *Slice[T]) ForEach(consumer func(T)) {
	slice.RLock()
	defer slice.RUnlock()

	for _, elem := range slice.elements {
		consumer(elem)
	}
}

func (slice *Slice[T]) Find(predicate func(T) bool) (T, bool) {
	var res T
	slice.RLock()
	defer slice.RUnlock()

	for _, elem := range slice.elements {
		if predicate(elem) {
			return elem, true
		}
	}

	return res, false
}

// Thread safe map. Any operation is guaranteed to be thread-safe.
type Map[K comparable, V any] struct {
	sync.RWMutex
	content map[K]V
}

func NewMap[K comparable, V any]() Map[K, V] {
	return Map[K, V]{
		content: make(map[K]V),
	}
}

// Returns the number of (key, value) pairs in the map.
func (m *Map[K, V]) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.content)
}
func (m *Map[K, V]) Entries() map[K]V {
	res := make(map[K]V)
	m.ForEach(func(k K, v V) {
		res[k] = v
	})
	return res
}

// Add an element to the map. Thread-safe.
// If there is already an entry in the map for the key
// the entry is replaced with the new value.
func (m *Map[K, V]) Add(key K, value V) {
	m.Lock()
	defer m.Unlock()
	m.content[key] = value
}

// Alias for Add
func (m *Map[K, V]) Set(key K, value V) {
	m.Add(key, value)
}

// Alias for Add
func (m *Map[K, V]) Update(key K, value V) {
	m.Add(key, value)
}

// Delete an element from the map. Thread-safe.
// If no element is present, it's a no-op.
func (m *Map[K, V]) Delete(key K) {
	m.Lock()
	defer m.Unlock()
	delete(m.content, key)
}

// Get an element from the map. The second value is
// false if the key is not in the map.
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.content[key]
	return value, ok
}

func (m *Map[K, V]) GetOrDefault(key K, defaultValue V) V {
	v, ok := m.Get(key)
	if !ok {
		return defaultValue
	}
	return v
}

// Warning does not accept self modification
func (m *Map[K, V]) ForEach(consumer func(K, V)) {
	m.RLock()
	defer m.RUnlock()

	for key, value := range m.content {
		consumer(key, value)
	}
}

// Thread-safe set.
type Set[T comparable] struct {
	underlyingMap Map[T, struct{}]
}

// Returns a new Set.
func NewSet[T comparable]() Set[T] {
	return Set[T]{
		underlyingMap: NewMap[T, struct{}](),
	}
}

// Add t to the set.
func (s *Set[T]) Add(t T) {
	s.underlyingMap.Add(t, struct{}{})
}

// Returns true iff the set contains the provided element.
func (s *Set[T]) Contains(t T) bool {
	_, ok := s.underlyingMap.Get(t)
	return ok
}

// Remove element from the set
func (s *Set[T]) Remove(t T) {
	s.underlyingMap.Delete(t)
}

func (s *Set[T]) Values() datastructures.Set[T] {
	res := datastructures.Set[T](s.underlyingMap.Entries())
	return res
}

func (s *Set[T]) Size() uint {
	return uint(len(s.underlyingMap.Entries()))
}
