package pool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type ResetableStruct struct {
	i int
	s string
}

func (rs *ResetableStruct) Reset() {
	rs.i = 0
	rs.s = ""
}

type NonPointerResetableStruct struct{}

func (nrs NonPointerResetableStruct) Reset() {}

func TestEmptyPool_Get(t *testing.T) {
	pool := New(func() *ResetableStruct {
		return &ResetableStruct{}
	})
	s := pool.Get()
	require.NotNil(t, s)
}

func TestPool_Reset(t *testing.T) {
	pool := New(func() *ResetableStruct {
		return &ResetableStruct{}
	})
	e := &ResetableStruct{i: 2, s: "str"}
	pool.Put(e)
	thisOne := pool.Get()
	require.Empty(t, thisOne.i)
	require.Empty(t, thisOne.s)
}

func TestPool_PutNil(t *testing.T) {
	var nilStruct *ResetableStruct
	pool := New(func() *ResetableStruct {
		return &ResetableStruct{}
	})
	require.NotPanics(t, func() {
		pool.Put(nilStruct)
	})
}

func TestPool_NewFuncNil(t *testing.T) {
	t.Run("pointer type", func(t *testing.T) {
		p := New[*ResetableStruct](nil)
		rs := p.Get()
		require.Zero(t, rs)
		require.NotPanics(t, func() {
			p.Put(rs)
		})
	})

	t.Run("value type", func(t *testing.T) {
		p := New[NonPointerResetableStruct](nil)
		rs := p.Get()
		require.Zero(t, rs)
	})
}

func TestEmptyNonPointerPool_Get(t *testing.T) {
	pool := New(func() NonPointerResetableStruct {
		return NonPointerResetableStruct{}
	})
	e := pool.Get()
	require.Empty(t, e)
}

func TestNonPointerPool_Put(t *testing.T) {
	pool := New(func() NonPointerResetableStruct {
		return NonPointerResetableStruct{}
	})
	e := NonPointerResetableStruct{}
	pool.Put(e)
	thisOne := pool.Get()
	require.Empty(t, thisOne)
}
