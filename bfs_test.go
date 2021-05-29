package promise

import (
	"testing"
	"time"
)

type Node int

var allNeighbors = map[Node][]Node {
	1: [2, 3],
	2: [1, 3, 5],
	3: [1, 2],
	4: [5],
	5: [2, 4],
}

func Neighbors(n Node) chan Node {
	c := make(chan Node)
	defer close(c)
	for _, nbr := range allNeighbors[n] {
		time.Sleep(time.Second)
		c <- nbr
	}
}

func TestBFS(t *testing.T) {
	root := Node(1)
	queue := []Node {root}
}

