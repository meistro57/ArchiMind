package qdrant

import (
	"context"
	"errors"
	"sync"
)

var ErrWorkerPoolClosed = errors.New("qdrant worker pool is closed")

type queryJob struct {
	ctx        context.Context
	collection string
	vectorName string
	vector     []float64
	limit      int
	resultCh   chan queryResult
}

type queryResult struct {
	points []SearchPoint
	err    error
}

type WorkerPool struct {
	client *Client
	jobs   chan queryJob
	done   chan struct{}
	wg     sync.WaitGroup
}

func NewWorkerPool(client *Client, workers int, queueSize int) *WorkerPool {
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = workers * 4
	}

	pool := &WorkerPool{
		client: client,
		jobs:   make(chan queryJob, queueSize),
		done:   make(chan struct{}),
	}

	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.runWorker()
	}

	return pool
}

func (w *WorkerPool) Query(ctx context.Context, collection string, vectorName string, vector []float64, limit int) ([]SearchPoint, error) {
	resultCh := make(chan queryResult, 1)
	job := queryJob{
		ctx:        ctx,
		collection: collection,
		vectorName: vectorName,
		vector:     vector,
		limit:      limit,
		resultCh:   resultCh,
	}

	select {
	case <-w.done:
		return nil, ErrWorkerPoolClosed
	default:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-w.done:
		return nil, ErrWorkerPoolClosed
	case w.jobs <- job:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-w.done:
		return nil, ErrWorkerPoolClosed
	case result := <-resultCh:
		return result.points, result.err
	}
}

func (w *WorkerPool) Close() {
	select {
	case <-w.done:
		return
	default:
		close(w.done)
	}
	w.wg.Wait()
}

func (w *WorkerPool) runWorker() {
	defer w.wg.Done()

	for {
		select {
		case <-w.done:
			return
		case job := <-w.jobs:
			points, err := w.client.Query(job.ctx, job.collection, job.vectorName, job.vector, job.limit)
			job.resultCh <- queryResult{points: points, err: err}
		}
	}
}
