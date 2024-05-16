package yugabyte

import (
	"time"
)

// Option - Option type
type Option func(*Yugabyte)

// MaxOpenConnections - Defines maximum open connections for Yugabyteql db
func MaxOpenConnections(size int) Option {
	return func(c *Yugabyte) {
		c.maxOpenConnections = size
	}
}

// MaxIdleConnections - Defines maximum idle connections for Yugabyteql db
func MaxIdleConnections(c int) Option {
	return func(p *Yugabyte) {
		p.maxIdleConnections = c
	}
}

// MaxConnectionIdleTime - Defines maximum connection idle for Yugabyteql db
func MaxConnectionIdleTime(d time.Duration) Option {
	return func(p *Yugabyte) {
		p.maxConnectionIdleTime = d
	}
}

// MaxConnectionLifeTime - Defines maximum connection lifetime for Yugabyteql db
func MaxConnectionLifeTime(d time.Duration) Option {
	return func(p *Yugabyte) {
		p.maxConnectionLifeTime = d
	}
}

func MaxDataPerWrite(v int) Option {
	return func(c *Yugabyte) {
		c.maxDataPerWrite = v
	}
}

func WatchBufferSize(v int) Option {
	return func(c *Yugabyte) {
		c.watchBufferSize = v
	}
}

func MaxRetries(v int) Option {
	return func(c *Yugabyte) {
		c.maxRetries = v
	}
}
