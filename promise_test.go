package promise

import (
	"testing"
	"github.com/stretchr/testify/require"
	"errors"
)

func TestPromiseBasic(t *testing.T) {
	p := NewPromise()
	p.Fulfill(1)
	val, err := p.Await()
	require.NoError(t, err)
	require.Equal(t, val, 1)

	p = NewPromise()
	p.Error(errors.New("ERROR"))
	val, err = p.Await()
	require.Error(t, err)
	require.Nil(t, val)
}

func TestPromiseConcurrent(t *testing.T) {
	p := NewPromise()
	trigger := make(chan struct{})
	done := make(chan struct{})
	var val PromisedValue
	var err error
	go func() {
		<-trigger
		p.Fulfill(2)
	}()
	go func() {
		val, err = p.Await()
		close(done)
	}()
	close(trigger)
	<-done
	require.Equal(t, val, 2)
	require.NoError(t, err)
}

func TestPromiseThen(t *testing.T) {
	p := NewPromise()
	trigger := make(chan struct{})
	go func() {
		<-trigger
		p.Fulfill(8)
	}()
	childPromise := p.Then(func(val PromisedValue) (PromisedValue, error) {
		return []PromisedValue{val}, nil
	})
	close(trigger)
	val, err := childPromise.Await()
	require.Equal(t, val, []PromisedValue{8})
	require.NoError(t, err)	
}
