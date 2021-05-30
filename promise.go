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

func (p *MultiPromise) isDone() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// can be called multiple times before done to populate the vals array
// attempts to fulfill after error or Completed are ignored.
func (p *MultiPromise) Fulfill(val PromisedValue) {
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.err != nil || p.isDone() {
		return
	}
	p.vals = append(p.vals, val)
	p.cv.Broadcast()
}

// safe to be called multiple times, including after error.
func (p *MultiPromise) Complete() {
	defer p.cv.Broadcast()
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.isDone() {
		return
	}
	close(p.done)
}

// only the first error matters. errors after the first or after Completed are ignored.
func (p *MultiPromise) Error(err error) {
	if err == nil {
		panic("cannot set err==nil")
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.err != nil || p.isDone() {
		return
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
func (p *MultiPromise) Receive(valCallback func(PromisedValue), errCallback func(error)) {
	p.mux.Lock()
	defer p.mux.Unlock()
	nextIndex := 0
	for {
		// processing the stuff already in p.vals
		for nextIndex < len(p.vals) {
			valCallback(p.vals[nextIndex])
			nextIndex++
		}
		// checks error status
		if p.err != nil {
			errCallback(p.err)
		}
		// checks whether p is done
		if p.isDone() {
			return
		}
		// if not done, wait for a signal
		p.cv.Wait()
	}
}

// example:  input [-1, 0, 1]. first map is 2/x. second map is x+1 and divide by zero error -> 0.
// expected output: [-1, 0, 3]. actual output: [-1, 0]
// decision: callback cannot catch errors.
func (p *MultiPromise) Map(
	valCallback func(PromisedValue) (PromisedValue, error),
	errCallback func(error) error,
) *MultiPromise {
	childPromise := NewMultiPromise()
	go func() {
		defer childPromise.Complete()
		// if it errors out at any point, stop processing the rest of the vals in parent promise.
		earlyError := false
		p.Receive(func(val PromisedValue) {
			if earlyError {
				return
			}
			val, err := valCallback(val)
			if err != nil {
				childPromise.Error(err)
				earlyError = true
			} else {
				childPromise.Fulfill(val)
			}
		}, func(err error) {
			if earlyError {
				return
			}
			err = errCallback(err)
			childPromise.Error(err)
		})
	}()
	return childPromise
}

func (p *MultiPromise) Then(callback func(PromisedValue) (PromisedValue, error)) *MultiPromise {
	return p.Map(func(val PromisedValue) (PromisedValue, error) {
		return callback(val)
	}, func (err error) error {
		return err
	})
}

func (p *MultiPromise) OnError(callback func(error) error) *MultiPromise {
	return p.Map(func(val PromisedValue) (PromisedValue, error) {
		return val, nil
	}, func(err error) error {
		return callback(err)
	})
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
