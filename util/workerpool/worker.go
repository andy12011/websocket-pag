package workerpool

import (
	"context"
)

func newWorker(ctx context.Context, id int, dispatcher *dispatcher) *worker {
	return &worker{
		id:             id,
		quit:           make(chan bool),
		taskDepositary: dispatcher,
		ctx:            ctx,
	}
}

type worker struct {
	id             int         // 工人編號
	quit           chan bool   // 關閉執行緒
	taskDepositary *dispatcher // 工作倉庫
	ctx            context.Context
}

func (w *worker) start() {
	go func() {
		for {
			select {
			case task := <-w.taskDepositary.taskStorage:
				resp, err := w.processTask(task)
				if err != nil || resp == nil {
					continue
				}

				task.CallBack(resp)

			case <-w.quit:
				return
			}
		}
	}()
}

func (w *worker) close() {
	w.quit <- true
}

func (w *worker) processTask(task Task) (interface{}, error) {
	resp, err := task.Exec()
	return resp, err
}
