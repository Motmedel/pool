package pool

import (
	"container/list"
	"fmt"
	motmedelErrors "github.com/Motmedel/utils_go/pkg/errors"
	"sync"
)

type Pool[T any] struct {
	NumMax      int
	MakeElement func() (T, error)
	PushFront   bool

	numActive int
	condition *sync.Cond
	elements  *list.List
	mutex     *sync.Mutex
}

func New[T any](numMax int, pushFront bool, fn func() (T, error)) *Pool[T] {
	mutex := new(sync.Mutex)
	return &Pool[T]{
		NumMax:      numMax,
		MakeElement: fn,
		PushFront:   pushFront,
		mutex:       mutex,
		elements:    list.New(),
		condition:   sync.NewCond(mutex),
	}
}

func (pool *Pool[T]) Get() (T, error) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	for pool.elements.Len() == 0 && pool.numActive >= pool.NumMax {
		pool.condition.Wait()
	}

	var zero T

	if pool.elements.Len() > 0 {
		untypedElement := pool.elements.Remove(pool.elements.Front())
		typedElement, ok := untypedElement.(T)
		if !ok {
			return zero, motmedelErrors.NewWithTrace(motmedelErrors.ErrConversionNotOk, untypedElement)
		}

		return typedElement, nil
	}

	element, err := pool.MakeElement()
	if err != nil {
		return zero, fmt.Errorf("make element: %w", err)
	}

	pool.numActive++

	return element, nil
}

func (pool *Pool[T]) Put(element T, err error) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if err != nil {
		pool.numActive--
	} else {
		if pool.PushFront {
			pool.elements.PushFront(element)
		} else {
			pool.elements.PushBack(element)
		}
	}

	pool.condition.Signal()
}

func (pool *Pool[T]) Len() int {
	return pool.elements.Len()
}
