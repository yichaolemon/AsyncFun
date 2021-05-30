package promise

import (
	"testing"
	"time"
	"fmt"
)

type Node int

var allNeighbors = map[Node][]Node {
	1: {2, 3},
	2: {1, 3, 5},
	3: {1, 2},
	4: {5},
	5: {2, 4},
}

func Neighbors(n Node) chan Node {
	c := make(chan Node)
	defer close(c)
	for _, nbr := range allNeighbors[n] {
		time.Sleep(time.Millisecond)
		c <- nbr
	}
	return c
}

func TestBFS(t *testing.T) {
	allNodes := NewMultiPromise()
	root := Node(1)
	nodesInLevel := uint32(1)
	promise := NewMultiPromise()
	promise.Fulfill(root)
	allNodes.Fulfill(root)
	promise.Complete()
	for nodesInLevel > 0 {
		nodesInLevel = 0
		nextLevel := NewMultiPromise()
		promise.Then(func(val PromisedValue) (PromisedValue, error) {
			n := val.(Node)
			for neighbor := range Neighbors(n) {
				nextLevel.Fulfill(neighbor)
				nodesInLevel++
				allNodes.Fulfill(neighbor)
			}
			return nil, nil
		})
		promise.Await()
		promise = nextLevel
	}
	allNodes.Then(func(val PromisedValue) (PromisedValue, error) {
		fmt.Printf("%v\n", val.(Node))
		return nil, nil
	})
	allNodes.Await()
	t.Fail()
}

