package backend

import (
	"math/rand"
	"time"

	"github.com/jdamick/tup/config"
)

type Manager struct {
	backends []Backend
}

type Backend struct {
	config.Backend
	LastUsed time.Time
}

func NewManager(conf *config.Config) *Manager {
	backends := []Backend{}
	conf.Log().Infof("Initializing backends =*=*=")
	for _, b := range conf.Backends {
		backends = append(backends, Backend{Backend: b, LastUsed: time.Now()})
		conf.Log().Infof("Backend: %v", b)
	}

	return &Manager{backends: backends}
}

func (b *Manager) Backend() Backend {
	idx := rand.Intn(len(b.backends))
	return b.backends[idx]
}
