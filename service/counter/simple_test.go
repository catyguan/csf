package counter

import (
	"bytes"
	"context"
	"testing"

	"github.com/catyguan/csf/core"
	"github.com/stretchr/testify/assert"
)

func createCounter() (Counter, *CounterService) {
	s := NewCounterService()
	si := core.NewSimpleServiceInvoker(s)
	return NewCounter(si), s
}

func TestBase(t *testing.T) {
	c, _ := createCounter()
	ctx := context.Background()

	v, err := c.GetValue(ctx, "test")
	assert.NoError(t, err)
	assert.True(t, 0 == v)
	v, err = c.AddValue(ctx, "test", 1)
	assert.NoError(t, err)
	assert.True(t, 1 == v)
	v, err = c.AddValue(ctx, "test", 10)
	assert.NoError(t, err)
	assert.True(t, 11 == v)
	v, err = c.GetValue(ctx, "test")
	assert.NoError(t, err)
	assert.True(t, 11 == v)
}

func TestSnapshot(t *testing.T) {
	ctx := context.Background()

	c1, s1 := createCounter()
	v, err := c1.AddValue(ctx, "test", 123)
	assert.NoError(t, err)
	assert.True(t, v == 123)

	buf := bytes.NewBuffer(make([]byte, 0))
	err = s1.CreateSnapshot(ctx, buf)
	assert.NoError(t, err)

	c2, s2 := createCounter()
	err = s2.ApplySnapshot(ctx, buf)
	assert.NoError(t, err)
	v, err = c2.GetValue(ctx, "test")
	assert.NoError(t, err)
	assert.True(t, v == 123)
}
