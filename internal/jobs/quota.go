package jobs

import (
	"fmt"
	"sync"
)

type Quota struct {
	mu       sync.Mutex
	limit    int64
	reserved int64
	holders  map[string]int64
}

func NewQuota(limit int64) *Quota {
	return &Quota{
		limit:   limit,
		holders: map[string]int64{},
	}
}

func (q *Quota) Reserve(jobID string, bytes int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.reserved+bytes > q.limit {
		return fmt.Errorf("quota exceeded")
	}
	q.reserved += bytes
	q.holders[jobID] += bytes
	return nil
}

func (q *Quota) Release(jobID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.reserved -= q.holders[jobID]
	delete(q.holders, jobID)
	if q.reserved < 0 {
		q.reserved = 0
	}
}
