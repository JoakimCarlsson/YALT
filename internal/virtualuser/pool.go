package virtualuser

import (
	"github.com/joakimcarlsson/yalt/internal/http"
	"sync"
)

// UserPool represents a pool of VirtualUsers.
type UserPool struct {
	pool *sync.Pool
}

// CreatePool creates a new UserPool.
func CreatePool(
	size int,
	scriptContent []byte,
) (*UserPool, error) {
	client := http.NewClient()

	pool := &sync.Pool{
		New: func() interface{} {
			vu, err := CreateVu(client, scriptContent)
			if err != nil {
				panic(err)
			}
			return vu
		},
	}

	for i := 0; i < size; i++ {
		pool.Put(pool.New())
	}

	return &UserPool{
		pool: pool,
	}, nil
}

// Fetch retrieves a VirtualUser from the pool.
func (p *UserPool) Fetch() *VirtualUser {
	return p.pool.Get().(*VirtualUser)
}

// Return returns a VirtualUser to the pool.
func (p *UserPool) Return(user *VirtualUser) {
	p.pool.Put(user)
}
