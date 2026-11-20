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
	pool := New[*ResetableStruct]()
	s := pool.Get()
	require.NotNil(t, s)
}

func TestPool_Reset(t *testing.T) {
	pool := New[*ResetableStruct]()
	e := &ResetableStruct{i: 2, s: "str"}
	pool.Put(e)
	thisOne := pool.Get()
	require.Empty(t, thisOne.i)
	require.Empty(t, thisOne.s)
}

func TestPool_PutNil(t *testing.T) {
	var nilStruct *ResetableStruct
	pool := New[*ResetableStruct]()
	require.NotPanics(t, func() {
		pool.Put(nilStruct)
	})
}

func TestEmptyNonPointerPool_Get(t *testing.T) {
	pool := New[NonPointerResetableStruct]()
	e := pool.Get()
	require.Empty(t, e)
}

func TestNonPointerPool_Put(t *testing.T) {
	pool := New[NonPointerResetableStruct]()
	e := NonPointerResetableStruct{}
	pool.Put(e)
	thisOne := pool.Get()
	require.Empty(t, thisOne)
}
