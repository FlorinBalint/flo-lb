package algos

import (
	"log"

	"golang.org/x/exp/constraints"
)

type Comparator[T any] interface {
	Less(a, b T) bool
}

type OrderedComparator[T constraints.Ordered] struct{}

func (OrderedComparator[T]) Less(a, b T) bool {
	return a < b
}

type ReversedComparator[T constraints.Ordered] struct{}

func (ReversedComparator[T]) Less(a, b T) bool {
	return b < a
}

type keyValue[K comparable, V any] struct {
	key   K
	value V
}
type AdressablePQ[K comparable, V any] struct {
	comp    Comparator[V]
	keyMap  map[K]int
	entries []keyValue[K, V]
}

func (addrPQ *AdressablePQ[K, V]) swap(i, j int) {
	aux := addrPQ.entries[i]
	addrPQ.entries[i] = addrPQ.entries[j]
	addrPQ.keyMap[addrPQ.entries[i].key] = i
	addrPQ.entries[j] = aux
	addrPQ.keyMap[aux.key] = j
}

func leftChild(index int) int {
	return 2*index + 1
}

func rightChild(index int) int {
	return 2*index + 2
}

func parentNode(index int) int {
	if index == 0 {
		return 0
	}
	return (index - 1) / 2
}

func (addrPQ *AdressablePQ[K, V]) heapifyDown(index int) {
	left := leftChild(index)
	right := rightChild(index)
	max := index

	if left < len(addrPQ.entries) &&
		addrPQ.comp.Less(addrPQ.entries[max].value, addrPQ.entries[left].value) {
		max = left
	}

	if right < len(addrPQ.entries) &&
		addrPQ.comp.Less(addrPQ.entries[max].value, addrPQ.entries[right].value) {
		max = right
	}

	if max != index {
		addrPQ.swap(index, max)
		addrPQ.heapifyDown(max)
	}
}

func (addrPQ *AdressablePQ[K, V]) bubbleUp(index int) {
	parent := parentNode(index)
	for parent != index {
		if addrPQ.comp.Less(
			addrPQ.entries[parent].value,
			addrPQ.entries[index].value,
		) {
			addrPQ.swap(index, parent)
			index = parent
			parent = parentNode(index)
		} else {
			break
		}
	}
}

func (addrPQ *AdressablePQ[K, V]) Push(key K, value V) bool {
	if _, ok := addrPQ.keyMap[key]; ok {
		log.Printf("Tried to add an entry that already exists %v", value)
		return false
	}
	entry := keyValue[K, V]{key, value}
	addrPQ.entries = append(addrPQ.entries, entry)
	addrPQ.keyMap[entry.key] = len(addrPQ.entries) - 1
	addrPQ.bubbleUp(len(addrPQ.entries) - 1)
	return true
}

func (addrPQ *AdressablePQ[K, V]) repairIndex(idx int) {
	if parent := parentNode(idx); parent != idx &&
		addrPQ.comp.Less(
			addrPQ.entries[parent].value,
			addrPQ.entries[idx].value) {
		addrPQ.bubbleUp(idx)
	} else {
		addrPQ.heapifyDown(idx)
	}
}

func (addrPQ *AdressablePQ[K, V]) Remove(key K) bool {
	if idx, ok := addrPQ.keyMap[key]; !ok {
		return false
	} else {
		addrPQ.swap(idx, len(addrPQ.entries)-1)
		addrPQ.entries = addrPQ.entries[:len(addrPQ.entries)-1]
		delete(addrPQ.keyMap, key)
		if len(addrPQ.entries) > 0 && idx < len(addrPQ.entries) {
			addrPQ.repairIndex(idx)
		}
	}
	return false
}

func (addrPQ *AdressablePQ[K, V]) Emplace(key K, newValue V) bool {
	if idx, ok := addrPQ.keyMap[key]; !ok {
		return false
	} else {
		newEntry := keyValue[K, V]{key, newValue}
		addrPQ.entries[idx] = newEntry
		addrPQ.repairIndex(idx)
	}
	return true
}

func (addrPQ *AdressablePQ[K, V]) Get(key K) V {
	if idx, ok := addrPQ.keyMap[key]; ok {
		return addrPQ.entries[idx].value
	}
	var v V // Return default value
	return v
}

func (addrPQ *AdressablePQ[K, V]) Values() []V {
	var vals []V
	for _, entry := range addrPQ.entries {
		vals = append(vals, entry.value)
	}
	return vals
}

func (addrPQ *AdressablePQ[K, V]) Top() V {
	if len(addrPQ.entries) > 0 {
		return addrPQ.entries[0].value
	}
	var v V
	return v
}

func (addrPQ *AdressablePQ[K, V]) Pop() V {
	if len(addrPQ.entries) > 0 {
		res := addrPQ.entries[0]
		addrPQ.swap(0, len(addrPQ.entries)-1)
		addrPQ.entries = addrPQ.entries[:len(addrPQ.entries)-1]
		delete(addrPQ.keyMap, res.key)
		addrPQ.heapifyDown(0)
		return res.value
	}
	var v V
	return v
}

func (addrPQ *AdressablePQ[K, V]) Size() int {
	return len(addrPQ.entries)
}

func (addrPQ *AdressablePQ[K, V]) Empty() bool {
	return len(addrPQ.entries) == 0
}

func NewPQ[K comparable, V constraints.Ordered]() *AdressablePQ[K, V] {
	return NewPQWithComparator[K, V](OrderedComparator[V]{})
}

func NewPQWithComparator[K comparable, V any](comp Comparator[V]) *AdressablePQ[K, V] {
	return &AdressablePQ[K, V]{
		comp:   comp,
		keyMap: make(map[K]int),
	}
}
