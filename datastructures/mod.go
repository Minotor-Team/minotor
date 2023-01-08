package datastructures

// Return a slice of elements that satisfy the predicate
func Filter[T comparable](slice []T, predicate func(t T) bool) []T {
	res := make([]T, 0)
	for _, elem := range slice {
		if predicate(elem) {
			res = append(res, elem)
		}
	}
	return res
}

// Return a slice of elements in 'slice' to which we applied 'mapping'
func Map[T any, S any](slice []T, mapping func(t T) S) []S {
	res := make([]S, len(slice))
	for idx, elem := range slice {
		res[idx] = mapping(elem)
	}
	return res
}

// Set structures. Wraps map[T]struct{}
type Set[T comparable] map[T]struct{}

// Creates a new empty set
func EmptySet[T comparable]() Set[T] {
	return make(map[T]struct{})
}

// Add an element to the set
func (s Set[T]) Add(t T) {
	s[t] = struct{}{}
}

// Returns true iff s contains t
func (s Set[T]) Contains(t T) bool {
	_, ok := s[t]
	return ok
}

// Remove element t from set s
func (s Set[T]) Remove(t T) {
	delete(s, t)
}

// Returns the number of elements in the set s
func (s Set[T]) Size() int {
	return len(s)
}

// Returns an array of elements in s
func (s Set[T]) ToArray() []T {
	res := make([]T, s.Size())
	i := 0
	for elem := range s {
		res[i] = elem
		i++
	}
	return res
}

// Add all the provided elements to the set s
func (s Set[T]) AddAll(t ...T) {
	for _, elem := range t {
		s.Add(elem)
	}
}
