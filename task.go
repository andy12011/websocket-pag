package ws

import (
	"golang.org/x/xerrors"

	"github.com/andy12011/websocket-pag/util/workerpool"
)

type Task interface {
	Exec() (interface{}, error)
	CallBack(interface{}) error
}

type TaskHandleFunc func(client *WSBrokerClient) (interface{}, error)
type task struct {
	Client *WSBrokerClient
	Handle TaskHandleFunc
}

func NewWSBrokerTask(client *WSBrokerClient, handle TaskHandleFunc) workerpool.Task {
	return &task{
		Client: client,
		Handle: handle,
	}
}

func (t *task) Exec() (interface{}, error) {
	if t.Client.Ready() {
		return t.Handle(t.Client)
	}

	return nil, nil
}

func (t *task) CallBack(resp interface{}) error {
	switch v := resp.(type) {
	case TaskResp:
		if err := t.Client.WriteMessage(ServerMsg{
			Command: v.Command,
			Code:    v.Code,
			Message: v.Message,
			Data:    v.Data,
		}); err != nil {
			return xerrors.Errorf("Websocket 封包傳送失敗: %w", err)
		}
	case []TaskResp:
		for _, taskResp := range v {
			if err := t.Client.WriteMessage(ServerMsg{
				Command: taskResp.Command,
				Code:    taskResp.Code,
				Message: taskResp.Message,
				Data:    taskResp.Data,
			}); err != nil {
				return xerrors.Errorf("Websocket 封包傳送失敗: %w", err)
			}
		}
	}
	return nil
}
