package virtualuser

import (
	"fmt"
	"github.com/joakimcarlsson/yalt/internal/http"
)

// UserPool represents a pool of VirtualUsers.
type UserPool struct {
	pool chan *VirtualUser
}

// CreatePool creates a new UserPool.
func CreatePool(
	size int,
	scriptContent []byte,
) (*UserPool, error) {
	p := &UserPool{
		pool: make(chan *VirtualUser, size),
	}

	client := http.NewClient()

	for i := 0; i < size; i++ {
		vu, err := CreateVu(client, scriptContent)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual user: %w", err)
		}
		p.pool <- vu
	}

	return p, nil
}

// Fetch retrieves a VirtualUser from the pool.
func (p *UserPool) Fetch() *VirtualUser {
	return <-p.pool
}

// Return returns a VirtualUser to the pool.
func (p *UserPool) Return(user *VirtualUser) {
	p.pool <- user
}
