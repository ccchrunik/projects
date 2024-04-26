package scheduler

import (
	"cube/node"
	"cube/task"
	"log"
	"math"
)

const (
	// LIEB square ice constant
	// https://en.wikipedia.org/wiki/Lieb%27s_square_ice_constant
	LIEB = 1.53960071783900203869
)

type Scheduler interface {
	SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node
	Score(t task.Task, nodes []*node.Node) map[string]float64
	Pick(scores map[string]float64, candidates []*node.Node) *node.Node
}

type RoundRobin struct {
	Name       string
	LastWorker int
}

func (r *RoundRobin) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return nodes
}

func (r *RoundRobin) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)

	var newWorker int
	if r.LastWorker+1 < len(nodes) {
		newWorker = r.LastWorker + 1
		r.LastWorker++
	} else {
		newWorker = 0
		r.LastWorker = 0
	}

	for idx, node := range nodes {
		if idx == newWorker {
			nodeScores[node.Name] = 0.1
		} else {
			nodeScores[node.Name] = 1.0
		}
	}

	return nodeScores
}

func (r *RoundRobin) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	var bestNode *node.Node
	var lowestScore float64
	for idx, node := range candidates {
		if idx == 0 {
			bestNode = node
			lowestScore = scores[node.Name]
			continue
		}

		if scores[node.Name] < lowestScore {
			bestNode = node
			lowestScore = scores[node.Name]
		}
	}

	return bestNode
}

type Greedy struct {
	Name string
}

func (g *Greedy) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return selectCandidateNodes(t, nodes)
}

func (g *Greedy) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)

	for _, node := range nodes {
		cpuUsage, err := node.CalculateCpuUsage()
		if err != nil {
			log.Printf("error calculating CPU usage for node %s, skipping: %v", node.Name, err)
			continue
		}
		cpuLoad := calculateLoad(float64(*cpuUsage), math.Pow(2, 0.8))
		nodeScores[node.Name] = cpuLoad
	}
	return nodeScores
}

func (g *Greedy) Pick(candidates map[string]float64, nodes []*node.Node) *node.Node {
	minCpu := 0.00
	var bestNode *node.Node
	for idx, node := range nodes {
		if idx == 0 {
			minCpu = candidates[node.Name]
			bestNode = node
			continue
		}

		if candidates[node.Name] < minCpu {
			minCpu = candidates[node.Name]
			bestNode = node
		}
	}
	return bestNode
}

// Enhanced PVM (Parallel Virtual Machine) Algorithm
//
// Implementation of the E-PVM algorithm laid out in http://www.cnds.jhu.edu/pub/papers/mosix.pdf.
// The algorithm calculates the "marginal cost" of assigning a task to a machine. In the paper and
// in this implementation, the only resources considered for calculating a task's marginal cost are
// memory and cpu.
type Epvm struct {
	Name string
}

func (e *Epvm) SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	return selectCandidateNodes(t, nodes)
}

func (e *Epvm) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	nodeScores := make(map[string]float64)
	maxJobs := 4.0

	for _, node := range nodes {
		cpuUsage, err := node.CalculateCpuUsage()
		if err != nil {
			log.Printf("error calculating CPU usage for node %s, skipping: %v", node.Name, err)
			continue
		}
		cpuLoad := calculateLoad(*cpuUsage, math.Pow(2, 0.8))

		memoryAllocated := float64(node.Stats.MemUsedKb()) + float64(node.MemoryAllocated)
		memoryPercentAllocated := memoryAllocated / float64(node.Memory)

		newMemPercent := (calculateLoad(memoryAllocated+float64(t.Memory/1000), float64(node.Memory)))
		memCost := math.Pow(LIEB, newMemPercent) + math.Pow(LIEB, (float64(node.TaskCount+1))/maxJobs) - math.Pow(LIEB, memoryPercentAllocated) - math.Pow(LIEB, float64(node.TaskCount)/float64(maxJobs))
		cpuCost := math.Pow(LIEB, cpuLoad) + math.Pow(LIEB, (float64(node.TaskCount+1))/maxJobs) - math.Pow(LIEB, cpuLoad) - math.Pow(LIEB, float64(node.TaskCount)/float64(maxJobs))

		nodeScores[node.Name] = memCost + cpuCost
	}
	return nodeScores
}

func (e *Epvm) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	minCost := 0.00
	var bestNode *node.Node
	for idx, node := range candidates {
		if idx == 0 {
			minCost = scores[node.Name]
			bestNode = node
			continue
		}

		if scores[node.Name] < minCost {
			minCost = scores[node.Name]
			bestNode = node
		}
	}
	return bestNode
}

func selectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node {
	var candidates []*node.Node
	for node := range nodes {

		if checkDisk(t, nodes[node].Disk-nodes[node].DiskAllocated) {
			candidates = append(candidates, nodes[node])
		}

	}
	return candidates
}

func checkDisk(t task.Task, diskAvailable int64) bool {
	return t.Disk <= diskAvailable
}

func calculateLoad(usage float64, capacity float64) float64 {
	return usage / capacity
}
