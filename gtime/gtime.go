package gtime

import (
	"log"
	"sync"
	"time"
)

type GTime struct {
	labels map[string]time.Time
	m      sync.Mutex
}

func New() *GTime {
	g := &GTime{
		labels: make(map[string]time.Time),
	}
	return g
}

func (g *GTime) Start(label string) {
	g.m.Lock()
	defer g.m.Unlock()

	g.labels[label] = time.Now()
}

func (g *GTime) End(label string) {
	g.m.Lock()
	defer g.m.Unlock()

	start, exists := g.labels[label]
	if !exists {
		log.Println("[warning] label", label, "does not exist")
		return
	}
	delete(g.labels, label)
	s := time.Since(start)
	log.Println(label, "took:", s)
}
