package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisPool(t *testing.T) {
	pool := newRedisPool("localhost:6379", "")
	assert.Equal(t, 3, pool.MaxIdle)
	// need to cast to int, because the values are different types (int32 and int)
	// see issue https://github.com/stretchr/testify/issues/155
	assert.Equal(t, int(240000000000), int(pool.IdleTimeout)) // 240 seconds
}
