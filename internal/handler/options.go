package handler

import (
	"sync"
	"time"
)

type ProxyOptions func(*Proxy)

func WithPort(port int) ProxyOptions {
	return func(c *Proxy) {
		c.port = port
	}
}

func WithWorkerCount(workerCount int) ProxyOptions {
	return func(c *Proxy) {
		c.workers = workerCount
	}
}

func WithServers(endpoints ...string) ProxyOptions {
	return func(c *Proxy) {
		for it, address := range endpoints {
			c.servers[address] = &Server{
				Address:  address,
				Weight:   it,
				Alive:    false,
				LastPong: time.Now(),
				Mutex:    sync.Mutex{},
			}
		}
	}
}

func WithHeartbeat(heartbeat time.Duration) ProxyOptions {
	return func(c *Proxy) {
		c.heartbeat = heartbeat
	}
}

func WithDeathTime(deathtime time.Duration) ProxyOptions {
	return func(c *Proxy) {
		c.deathtime = deathtime
	}
}
