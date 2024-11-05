package trtest

import (
	"context"
	"github.com/meow-pad/persian/utils/gopool"
)

func newGoPool(maxGoNum int, queueSize int) (*GoPool, error) {
	pool := &GoPool{}
	goPool, err := gopool.NewGoroutinePool("io-pool",
		maxGoNum, queueSize, true)
	if err != nil {
		return nil, err
	}
	pool.GoroutinePool = goPool
	return pool, nil
}

type GoPool struct {
	*gopool.GoroutinePool
}

func (pool *GoPool) Start() error {
	return pool.GoroutinePool.Start()
}

func (pool *GoPool) Stop(ctx context.Context) error {
	return pool.GoroutinePool.Stop(ctx)
}
