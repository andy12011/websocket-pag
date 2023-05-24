package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/xerrors"

	"github.com/andy12011/websocket-pag/util/logger"
	"github.com/andy12011/websocket-pag/util/workerpool"
)

type StringChan chan string
type HandlerFunc func(ctx context.Context, reqData json.RawMessage) Resp
type HandlerFuncsMap map[string]HandlerFunc
type SendQueue chan []byte
type BroadcastData struct {
	Topic        string
	RawServerMsg []byte
}
type BroadcastQueue chan BroadcastData

var (
	once     sync.Once
	wsBroker *WSBroker
)

func newWSBroker(params NewParams) WSBrokerInterface {
	sysLog, err := logger.InitSysLog(params.ServiceName, params.ServiceLogLevel)
	if err != nil {
		panic(err)
	}
	once.Do(func() {
		wsBroker = &WSBroker{
			upgrade: websocket.Upgrader{
				// 如果有 cross domain 的需求，可加入這個，不檢查 cross domain
				CheckOrigin: func(r *http.Request) bool { return params.IsCrossDomain },
			},
			handlerFuncs:   make(HandlerFuncsMap),
			BroadcastQueue: make(BroadcastQueue, params.BroadcastBuf),
			sendTimeout:    time.Duration(params.SendTimeout) * time.Second,
			workerPool:     workerpool.NewWorkerPool(context.Background(), params.PoolBuf, params.PoolWorkers),
			sysLog:         sysLog,
		}

		go func() {
			for {
				select {
				case broadcastData := <-wsBroker.BroadcastQueue:
					for _, client := range wsBroker.GetAllClient() {
						client.broadcastQueue <- broadcastData
					}
				}
			}
		}()

		wsBroker.workerPool.Start()
	})

	return wsBroker
}

type WSBrokerInterface interface {
	UpgradeHttp(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	RegisterHandler(command string, handler HandlerFunc)

	Broadcast(topic string, data interface{}) error
	SendTopicWithHandleFunc(ctx context.Context, topic string, handleFunc TaskHandleFunc) error
	SendWithHandleFunc(ctx context.Context, handleFunc TaskHandleFunc) error
	SendWithHandleFuncByPlayerID(ctx context.Context, playerID string, handleFunc TaskHandleFunc) error

	GetAllClient() []*WSBrokerClient
	GetClientsByTopic(topic string) []*WSBrokerClient
	GetClientByID(clientID string) *WSBrokerClient
	GetClientByPlayerID(playerID string) []*WSBrokerClient

	SubscribeTopic(ctx context.Context, topic string) error
	UnsubscribeTopic(ctx context.Context, topic string) error
}

type WSBroker struct {
	ClientPool     sync.Map
	handlerFuncs   HandlerFuncsMap
	upgrade        websocket.Upgrader
	sendTimeout    time.Duration
	BroadcastQueue BroadcastQueue
	workerPool     workerpool.WorkerPoolInterface
	sysLog         logger.LoggerInterface
}

func (wsBroker *WSBroker) RegisterHandler(command string, handler HandlerFunc) {
	wsBroker.handlerFuncs[command] = handler
	log.Println(fmt.Sprintf("[註冊Command] %s", command))
}

func (wsBroker *WSBroker) UpgradeHttp(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// 升級連線
	ctx, cancel := context.WithCancel(ctx)
	client, err := newBrokerClient(ctx, wsBroker, w, r)
	if err != nil {
		return xerrors.Errorf("wsbroker.HandlerFunc.newBrokerClient err: %w", err)
	}

	wsBroker.addClientToPool(client)

	// 定義必要參數
	stop := make(chan struct{})

	// 連線中斷
	defer func() {
		client.Stop()
		cancel()
		wsBroker.sysLog.Debug(client.Ctx, "WS 斷線囉")
		wsBroker.removeClientFromPool(client)
	}()

	// 處理收到封包
	go func() {
		for {
			if err := wsBroker.receive(client); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					wsBroker.sysLog.Error(ctx, xerrors.Errorf("Websocket 異常關閉: %w", err))
				}

				stop <- struct{}{}
				return
			}
		}
	}()

	// 處理送出封包
	go func() {
		for {
			if err := wsBroker.send(client); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					wsBroker.sysLog.Error(ctx, xerrors.Errorf("Websocket 異常關閉: %w", err))
				}

				stop <- struct{}{}
				return
			}
		}
	}()

	// 處理訂閱Topic
	go func() {
		for {
			var logMap []string
			select {
			case topic := <-client.subscribeChan:
				client.subscribeTopicMap.Store(topic, true)
				client.subscribeTopicMap.Range(func(key, value interface{}) bool {
					logMap = append(logMap, key.(string))
					return true
				})
				fmt.Println("[", client.ID, "] 已訂閱 Topic : ", logMap)
			case topic := <-client.unsubscribeChan:
				client.subscribeTopicMap.Delete(topic)
				client.subscribeTopicMap.Range(func(key, value interface{}) bool {
					logMap = append(logMap, key.(string))
					return true
				})
				fmt.Println("[", client.ID, "] 已訂閱 Topic : ", logMap)
			case <-ctx.Done():
				return
			}
		}
	}()

	client.Start()

	// 監聽停止事件
	select {
	case <-stop:
		return nil
	}
}

func (wsBroker *WSBroker) receive(client *WSBrokerClient) error {
	// 確認context 狀態
	select {
	case <-client.Ctx.Done():
		return &websocket.CloseError{Code: websocket.CloseGoingAway, Text: "Ctx Cancel"}
	default:
	}

	client.conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// 讀取收到的訊息
	_, messageByte, err := client.conn.ReadMessage()
	if err != nil {
		return err
	}

	if string(messageByte) == "ping" {
		client.sendQueue <- []byte("pong")

		return nil
	}

	// 解析收到的訊息
	message, err := wsBroker.parseClientMsg(messageByte)
	if err != nil {
		return err
	}

	// 執行相對應的功能
	if f, ok := wsBroker.handlerFuncs[message.Command]; ok {
		go func() {
			funcResp := f(client.Ctx, message.Data)

			if err := client.WriteMessage(ServerMsg{
				Command: message.Command,
				Seq:     message.Seq,
				Code:    funcResp.Code,
				Message: funcResp.Message,
				Data:    funcResp.Data,
			}); err != nil {
				wsBroker.sysLog.Error(client.Ctx, err)
			}
		}()
	}

	return nil
}

func (wsBroker *WSBroker) Broadcast(topic string, data interface{}) error {
	rawServerMsg, err := wsBroker.marshalServerMsg(data)
	if err != nil {
		return xerrors.Errorf("wsbroker.HandlerFunc.rawServerMsg err: %w", err)
	}

	broadcastData := BroadcastData{
		Topic:        topic,
		RawServerMsg: rawServerMsg,
	}
	wsBroker.BroadcastQueue <- broadcastData

	return nil
}

func (wsBroker *WSBroker) SendWithHandleFunc(ctx context.Context, handleFunc TaskHandleFunc) error {
	clients := wsBroker.GetAllClient()

	for _, client := range clients {
		if err := wsBroker.workerPool.ReceiveTask(NewWSBrokerTask(client, handleFunc)); err != nil {
			return xerrors.Errorf("接收任務失敗 :%w", err)
		}
	}

	return nil
}

func (wsBroker *WSBroker) SendTopicWithHandleFunc(ctx context.Context, topic string, handleFunc TaskHandleFunc) error {
	clients := wsBroker.GetClientsByTopic(topic)

	for _, client := range clients {
		if err := wsBroker.workerPool.ReceiveTask(NewWSBrokerTask(client, handleFunc)); err != nil {
			return xerrors.Errorf("接收任務失敗 :%w", err)
		}
	}

	return nil
}

func (wsBroker *WSBroker) SendWithHandleFuncByPlayerID(ctx context.Context, playerID string, handleFunc TaskHandleFunc) error {
	clients := wsBroker.GetClientByPlayerID(playerID)

	for _, client := range clients {
		if err := wsBroker.workerPool.ReceiveTask(NewWSBrokerTask(client, handleFunc)); err != nil {
			return xerrors.Errorf("接收任務失敗: %w", err)
		}
	}

	return nil
}

func (wsBroker *WSBroker) parseClientMsg(buf []byte) (ClientMsg, error) {
	clientMsg := ClientMsg{}
	if err := json.Unmarshal(buf, &clientMsg); err != nil {
		return clientMsg, xerrors.Errorf("wsbroker.HandlerFunc.Unmarshal err: %w", err)
	}

	return clientMsg, nil
}

func (wsBroker *WSBroker) marshalServerMsg(data interface{}) ([]byte, error) {
	rawServerMsg, err := json.Marshal(data)
	if err != nil {
		return nil, xerrors.Errorf("wsbroker.HandlerFunc.rawServerMsg err: %w", err)
	}

	return rawServerMsg, nil
}

func (wsBroker *WSBroker) send(client *WSBrokerClient) error {
	select {
	case marshalServerMsg := <-client.sendQueue:
		if err := client.conn.WriteMessage(websocket.TextMessage, marshalServerMsg); err != nil {
			return xerrors.Errorf("wsbroker.HandlerFunc.marshalServerMsg: %w", err)
		}
	case broadcastData := <-client.broadcastQueue:
		if client.IsSubscribed(broadcastData.Topic) {
			if err := client.conn.WriteMessage(websocket.TextMessage, broadcastData.RawServerMsg); err != nil {
				return xerrors.Errorf("wsbroker.HandlerFunc.broadcastData: %w", err)
			}
		}
	case <-client.Ctx.Done():
		return &websocket.CloseError{Code: websocket.CloseGoingAway, Text: "Ctx Cancel"}
	}

	return nil
}

var count = 0

func (wsBroker *WSBroker) addClientToPool(client *WSBrokerClient) {
	count += 1
	fmt.Println("[", count, "]addClientToPool : ", client.ID, ", Player : ", client.playerID)

	wsBroker.ClientPool.Store(client.ID, client)
}

func (wsBroker *WSBroker) removeClientFromPool(client *WSBrokerClient) {
	count -= 1
	fmt.Println("[", count, "]removeClientFromPool : ", client.ID, ", Player : ", client.playerID)

	wsBroker.ClientPool.Delete(client.ID)
	_ = client.conn.Close()
}

func (wsBroker *WSBroker) GetClientByID(clientID string) *WSBrokerClient {
	if value, ok := wsBroker.ClientPool.Load(clientID); ok {
		return value.(*WSBrokerClient)
	} else {
		return nil
	}
}

func (wsBroker *WSBroker) GetAllClient() []*WSBrokerClient {
	var clients []*WSBrokerClient

	wsBroker.ClientPool.Range(func(key, value interface{}) bool {
		if client, ok := value.(*WSBrokerClient); ok {
			if client.Ready() {
				clients = append(clients, value.(*WSBrokerClient))
			}
		}

		return true
	})

	return clients
}

func (wsBroker *WSBroker) GetClientsByTopic(topic string) []*WSBrokerClient {
	var clients []*WSBrokerClient

	for _, client := range wsBroker.GetAllClient() {
		if client.IsSubscribed(topic) {
			clients = append(clients, client)
		}
	}

	return clients
}

func (wsBroker *WSBroker) GetClientByPlayerID(playerID string) []*WSBrokerClient {
	var clients []*WSBrokerClient

	for _, client := range wsBroker.GetAllClient() {
		if client.playerID == playerID {
			clients = append(clients, client)
		}
	}

	return clients
}

func (wsBroker *WSBroker) SubscribeTopic(ctx context.Context, topic string) error {
	clientID := ctx.Value("clientID").(string)

	client := wsBroker.GetClientByID(clientID)
	client.Subscribe(topic)

	return nil
}

func (wsBroker *WSBroker) UnsubscribeTopic(ctx context.Context, topic string) error {
	clientID := ctx.Value("clientID").(string)

	client := wsBroker.GetClientByID(clientID)
	client.Unsubscribe(topic)

	return nil
}
