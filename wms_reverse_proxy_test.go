package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisPool(t *testing.T) {
	pool := newRedisPool("localhost:6379", "")
	assert.Equal(t, 3, pool.MaxIdle)
	assert.Equal(t, 240000000000, pool.IdleTimeout) // 240 seconds
}
