package control

import (
	"math"
	"sync"
)

type Task func() error

type TaskItem struct {
	fn         func() error
	SlothIndex int64
	Currents   int64
}

// LoadBalancer The implementation of Least-Connections Load Balancer with Failure Drop.
type LoadBalancer struct {
	TaskQueue []*TaskItem
	sync.Mutex
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{}
}

func (lb *LoadBalancer) Len() int {
	return len(lb.TaskQueue)
}

// Add a new task to the [LoadBalancer]
// @param priority The smaller the value, the higher the priority.
func (lb *LoadBalancer) Add(task Task, priority int64) {
	if task == nil {
		panic("empty task is not allowed")
	}
	lb.TaskQueue = append(lb.TaskQueue, &TaskItem{fn: task, SlothIndex: priority, Currents: 0})
}

func (lb *LoadBalancer) Dispatch() {
	var candidate *TaskItem
	lb.Lock()
	var minWeighted int64 = math.MaxInt64
	for i := 0; i < lb.Len(); i++ {
		weighted := lb.TaskQueue[i].Currents * lb.TaskQueue[i].SlothIndex
		if weighted < minWeighted {
			minWeighted = weighted
			candidate = lb.TaskQueue[i]
		}
	}
	if candidate == nil || candidate.fn == nil {
		return
	}
	candidate.Currents++
	lb.Unlock()
	err := candidate.fn()
	lb.Lock()
	defer lb.Unlock()
	if err == nil {
		candidate.Currents--
	}
}
