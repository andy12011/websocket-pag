package workerpool

import (
	"context"

	"golang.org/x/xerrors"

	"gitlab.paradise-soft.com.tw/glob/ws/util/logger"
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
				if err != nil {
					if xerrors.Is(err, context.Canceled) {
						logger.SysLog().Debug(w.ctx, "context cancel")
					} else {
						// 如果是 context.Cancel 就算是正常離開，不用印 Error
						logger.SysLog().Error(w.ctx, xerrors.Errorf("worker #%d is done with error: %w", w.id, err))
					}
					continue
				}

				if resp == nil {
					continue
				}

				if err := task.CallBack(resp); err != nil {
					logger.SysLog().Error(w.ctx, xerrors.Errorf("worker #%d is done with error: %w", w.id, err))
				}
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
