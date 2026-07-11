package queue

import "context"

// Job represents a single item retrieved from the queue.
type Job struct {
	ID        string
	Type      string
	Payload   []byte
	Attempts  int
}

// QueueProvider abstracts the queue storage operations, allowing us to swap
// backends (In-memory, PostgreSQL SKIP LOCKED, AWS SQS, OCI Queue) seamlessly.
type QueueProvider interface {
	// Enqueue puts a payload onto the queue for asynchronous execution.
	Enqueue(ctx context.Context, jobType string, payload []byte) (string, error)

	// Dequeue pulls the next available job off the queue.
	Dequeue(ctx context.Context) (*Job, error)

	// Ack acknowledges successful completion of the job (removes it).
	Ack(ctx context.Context, jobID string) error

	// Nack releases the job back to the queue (retrying or sending to DLQ).
	Nack(ctx context.Context, jobID string, err error) error
}
