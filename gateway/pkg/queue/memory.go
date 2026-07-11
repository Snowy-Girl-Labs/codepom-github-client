package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type MemoryQueueJob struct {
	Job
	VisibleAfter time.Time
}

// MemoryProvider implements the QueueProvider interface using an in-memory queue.
type MemoryProvider struct {
	mu   sync.Mutex
	jobs []*MemoryQueueJob
	seq  int64
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		jobs: make([]*MemoryQueueJob, 0),
	}
}

func (m *MemoryProvider) Enqueue(ctx context.Context, jobType string, payload []byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.seq++
	id := fmt.Sprintf("mem_job_%d", m.seq)
	m.jobs = append(m.jobs, &MemoryQueueJob{
		Job: Job{
			ID:       id,
			Type:     jobType,
			Payload:  payload,
			Attempts: 0,
		},
		VisibleAfter: time.Now(),
	})
	return id, nil
}

func (m *MemoryProvider) Dequeue(ctx context.Context) (*Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for i, j := range m.jobs {
		if j.VisibleAfter.Before(now) {
			j.Attempts++
			// Set visibility timeout (e.g. lock it for 30s during processing)
			j.VisibleAfter = now.Add(30 * time.Second)

			// Return a copy of the job
			jobCopy := j.Job
			return &jobCopy, nil
		}
	}
	return nil, nil // No jobs available
}

func (m *MemoryProvider) Ack(ctx context.Context, jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, j := range m.jobs {
		if j.ID == jobID {
			// Remove the job from the list
			m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
			return nil
		}
	}
	return errors.New("job not found")
}

func (m *MemoryProvider) Nack(ctx context.Context, jobID string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, j := range m.jobs {
		if j.ID == jobID {
			if j.Attempts >= 3 {
				// Dead-letter simulation: remove if max retries exceeded
				for i, currentJob := range m.jobs {
					if currentJob.ID == jobID {
						m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
						break
					}
				}
				return fmt.Errorf("job %s exceeded max retries: %w", jobID, err)
			}
			// Make visible again immediately
			j.VisibleAfter = time.Now()
			return nil
		}
	}
	return errors.New("job not found")
}
