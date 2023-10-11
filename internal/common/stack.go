package common

type Stack[T any] struct {
	top    *stackNode[T]
	length int
}

type stackNode[T any] struct {
	value T
	prev  *stackNode[T]
}

func (s *Stack[T]) Pop() (T, bool) {
	if s.length == 0 || s.top == nil {
		var zero T
		return zero, false
	}

	value := s.top.value
	s.top = s.top.prev
	s.length--

	return value, true
}

func (s *Stack[T]) Push(values ...T) {
	for i := len(values) - 1; i >= 0; i-- {
		value := values[i]
		s.length++
		s.top = &stackNode[T]{
			value: value,
			prev:  s.top,
		}
	}
}
