package workerpool

import (
	"context"
)

// NewWorkerPool return a pool with workers
func NewWorkerPool(ctx context.Context, buf, numOfWorkers int) WorkerPoolInterface {
	wp := &workerPool{
		dispatcher: newDispatcher(buf),
		workers:    make([]*worker, 0, numOfWorkers),
	}

	for i := 0; i < numOfWorkers; i++ {
		wp.workers = append(wp.workers, newWorker(ctx, i+1, wp.dispatcher))
	}

	return wp
}

// WorkerPoolInterface will receive tasks and dispatch them to workers
type WorkerPoolInterface interface {
	Start()
	Close()
	ReceiveTask(task Task) error
}

type workerPool struct {
	dispatcher *dispatcher
	workers    []*worker

	allowTaskOverFlow bool
}

func (wp *workerPool) Start() {
	for _, worker := range wp.workers {
		worker.start()
	}
}

func (wp *workerPool) Close() {
	for _, worker := range wp.workers {
		worker.close()
	}
}

func (wp *workerPool) ReceiveTask(task Task) error {
	if err := wp.dispatcher.receiveTask(task); err != nil {
		return err
	}

	return nil
}
