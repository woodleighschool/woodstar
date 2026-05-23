package software

type orderedGroup[K comparable, V any] struct {
	byKey map[K]*V
	order []K
}

func newOrderedGroup[K comparable, V any]() orderedGroup[K, V] {
	return orderedGroup[K, V]{
		byKey: make(map[K]*V),
		order: make([]K, 0),
	}
}

func (g *orderedGroup[K, V]) get(key K, create func() V) *V {
	value := g.byKey[key]
	if value != nil {
		return value
	}
	created := create()
	value = &created
	g.byKey[key] = value
	g.order = append(g.order, key)
	return value
}

func (g orderedGroup[K, V]) values() []V {
	out := make([]V, 0, len(g.order))
	for _, key := range g.order {
		out = append(out, *g.byKey[key])
	}
	return out
}
