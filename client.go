package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/xerrors"
)

type WSBrokerClient struct {
	ID                string
	Ctx               context.Context
	playerID          string
	conn              *websocket.Conn
	subscribeTopicMap sync.Map
	subscribeChan     StringChan
	unsubscribeChan   StringChan
	sendQueue         SendQueue
	broadcastQueue    BroadcastQueue
	ready             bool
}

func newBrokerClient(ctx context.Context, wsBroker *WSBroker, w http.ResponseWriter, r *http.Request) (*WSBrokerClient, error) {
	conn, err := wsBroker.upgrade.Upgrade(w, r, nil)
	if err != nil {
		return nil, xerrors.Errorf("建立 Client WSBroker 失败: %w", err)
	}
	if ctx.Value("playerID") == nil {
		return nil, xerrors.Errorf("請在ctx帶入使用者的識別碼(playID)")
	}
	playerID, ok := ctx.Value("playerID").(string)
	if !ok {
		return nil, xerrors.Errorf("使用者的識別碼應為字串")
	}
	if playerID == "" {
		return nil, xerrors.Errorf("使用者的識別碼不可為空字串")
	}

	clientID := uuid.New().String()
	ctx = context.WithValue(ctx, "clientID", clientID)
	ctx = context.WithValue(ctx, "playerID", playerID)

	client := &WSBrokerClient{
		ID:              clientID,
		Ctx:             ctx,
		playerID:        playerID,
		conn:            conn,
		ready:           false,
		subscribeChan:   make(StringChan, 100),
		unsubscribeChan: make(StringChan, 100),
		sendQueue:       make(SendQueue, 1000),
		broadcastQueue:  make(BroadcastQueue, 1000),
	}

	return client, nil
}

func (c *WSBrokerClient) Start() {
	c.ready = true
}

func (c *WSBrokerClient) Stop() {
	c.ready = false
}

func (c *WSBrokerClient) Ready() bool {
	return c.ready
}

func (c *WSBrokerClient) GetPlayerID() string {
	return c.playerID
}

func (c *WSBrokerClient) Subscribe(topic string) {
	c.subscribeChan <- topic
}

func (c *WSBrokerClient) Unsubscribe(topic string) {
	c.unsubscribeChan <- topic
}

func (c *WSBrokerClient) IsSubscribed(topic string) bool {
	if _, ok := c.subscribeTopicMap.Load(topic); ok {
		return true
	} else {
		return false
	}
}

func (c *WSBrokerClient) WriteMessage(data interface{}) error {
	marshalServerMsg, err := json.Marshal(data)
	if err != nil {
		return xerrors.Errorf("json marshal err: %w", err)
	}

	c.sendQueue <- marshalServerMsg

	return nil
}
