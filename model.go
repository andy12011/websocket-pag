package ws

import (
	"context"
	"encoding/json"
)

type ClientMsg struct {
	Command string          `json:"command"`
	Seq     string          `json:"seq"`
	Data    json.RawMessage `json:"data"`
}

type ServerMsg struct {
	Command string      `json:"command"`
	Seq     string      `json:"seq"`
	Code    string         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Resp struct {
	Code    string
	Message string
	Data    interface{}
}

type TaskResp struct {
	Ctx context.Context
	Command string
	Code    string
	Message string
	Data    interface{}
}
