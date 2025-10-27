package arraylist

type ArrayList[T comparable] struct {
	items []T
}

func (al *ArrayList[T]) Len() int {
	return len(al.items)
}

func (al *ArrayList[T]) Push(item T) {
	al.items = append(al.items, item)
}

// Contains 1D checking
func (al *ArrayList[T]) Contains(against T) bool {
	for _, item := range al.items {
		if item == against {
			return true
		}
	}
	return false
}

func NewArrayList[T comparable](items []T) *ArrayList[T] {
	return &ArrayList[T]{}
}
