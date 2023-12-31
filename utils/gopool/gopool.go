package gopool

import "github.com/meow-pad/persian/utils/gopool"

func NewGoPool(pool *gopool.GoroutinePool) *GoPool {
	return &GoPool{
		inner: pool,
	}
}

type GoPool struct {
	inner *gopool.GoroutinePool
}

func (pool *GoPool) Submit(task func()) error {
	return pool.inner.Submit(task)
}
