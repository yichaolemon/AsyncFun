package promise

import (
	"sync"
)

type PromisedValue interface{}

type Promise struct {
	// lock protecting val and err
	mux sync.Mutex
	// nil until promise is done
	val PromisedValue
	err error
	// gets closed when value is popluated
	done chan struct{}
}

type MultiPromise struct {
	// lock protecting val and err
	mux sync.Mutex
	// will get populated until done 
	vals []PromisedValue
	cv *sync.Cond
	// first error causes everything else to fail
	err error
	// gets closed when value is popluated
	done chan struct{}
}

func NewMultiPromise() *MultiPromise {
	p := &MultiPromise{
		done: make(chan struct{}),
	}
	p.cv = sync.NewCond(&p.mux)

	return p
}

// can be called multiple times before done to populate the vals array
func (p *MultiPromise) Fulfill(val PromisedValue) {
	if val == nil {
		panic("cannot set val==nil")
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.err != nil {
		panic("cannot fulfill after error")
	}
	p.vals = append(p.vals, val)
	p.cv.Broadcast()
}

func (p *MultiPromise) Complete() {
	close(p.done)
	p.cv.Broadcast()
}

func (p *MultiPromise) Error(err error) {
	if err == nil {
		panic("cannot set err==nil")
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.err != nil {
		panic("cannot set err twice")
	}
	select {
	case <-p.done: 
		panic("cannot error out after promise already completed")
	default: 
	}
	p.err = err
	close(p.done)
	p.cv.Broadcast()
}

func (p *MultiPromise) Await() ([]PromisedValue, error) {
	<-p.done
	return p.vals, p.err
}

// receive calls callback on all values in p.vals in order, returns when p is done
// if the promise errors out, then the last callback would be error
func (p *MultiPromise) Receive(callback func(PromisedValue, error)) {
	p.mux.Lock()
	// processing the stuff already in p.vals
	for _, v := range p.vals {
		callback(v, nil)
	}
	// checks error status
	if p.err != nil {
		callback(nil, p.err)
		p.mux.Unlock()
		return
	}
	// checks whether p is done
	select {
	case <-p.done:
		p.mux.Unlock()
		return
	default:
	}
	// if not done, 
}

func (p *MultiPromise) Map(callback func(PromisedValue, error) (PromisedValue, error)) *Promise {
	childPromise := NewMultiPromise()
	go func() {



		vals, err := p.Await()
		val, err = callback(val, err)
		if err == nil {
			childPromise.Fulfill(val)
		} else {
			childPromise.Error(err)
		}
	}()
	return childPromise
}

func NewPromise() *Promise {
	return &Promise{
		done: make(chan struct{}),
	}
}

func (p *Promise) Fulfill(val PromisedValue) {
	if val == nil {
		panic("cannot set val==nil")
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.val != nil || p.err != nil {
		panic("cannot set val twice")
	}
	p.val = val
	close(p.done)
}

func (p *Promise) Error(err error) {
	if err == nil {
		panic("cannot set err==nil")
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.val != nil || p.err != nil {
		panic("cannot set err twice")
	}
	p.err = err
	close(p.done)
}

func (p *Promise) Await() (PromisedValue, error) {
	<-p.done
	return p.val, p.err
}

func (p *Promise) chain(callback func(PromisedValue, error) (PromisedValue, error)) *Promise {
	childPromise := NewPromise()
	go func() {
		val, err := p.Await()
		val, err = callback(val, err)
		if err == nil {
			childPromise.Fulfill(val)
		} else {
			childPromise.Error(err)
		}
	}()
	return childPromise
}

func (p *Promise) Then(callback func(PromisedValue) (PromisedValue, error)) *Promise {
	return p.chain(func(val PromisedValue, err error) (PromisedValue, error) {
		if err == nil {
			return callback(val)
		}
		return nil, err
	})
}

func (p *Promise) OnError(callback func(error) (PromisedValue, error)) *Promise {
	return p.chain(func(val PromisedValue, err error) (PromisedValue, error) {
		if err != nil {
			return callback(err)
		}
		return val, nil
	})
}
