# ws

新的 websocket 套件

## Installation
取得專案
```shell
$ go get -u -v -insecure github.com/andy12011/websocket-pag
```

匯入到你的專案中
```go
import "github.com/andy12011/websocket-pag"
```

## Quick start

服務端(Server)
```go
package main

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/andy12011/websocket-pag"
)

var (
	ginEngine *gin.Engine
	wsEngine  ws.WSBrokerInterface
)

func main() {
	// 產生http引擎與websocket引擎
	ginEngine = gin.Default()
	wsEngine = ws.Default()

	// 設定websocket溝通路徑
	registerWsHandler()

	// 升級http連線成為websocket
	ginEngine.GET("/ws", upgradeHTTP)

	// 啟動http引擎在8080port運作
	ginEngine.Run(":8080")
}

func upgradeHTTP(c *gin.Context) {
	ctx := setPlayerID(c)
	wsEngine.UpgradeHttp(ctx, c.Writer, c.Request)
}

func setPlayerID(ctx context.Context) context.Context {
	return context.WithValue(ctx, "playerID", "0d0d85d2-4e12-41c1-b34e-f8143e347d61")
}

func registerWsHandler() {
	helloHandler := func(ctx context.Context, reqData json.RawMessage) ws.Resp {
		return ws.Resp{
			Code:    "200",
			Message: "Hello!",
			Data:    "websocket test success",
		}
	}

	// 當收到 command = "/hello" 時，回應 "Hello!"
	wsEngine.RegisterHandler("/hello", helloHandler)
}
```

客戶端(Client)
* 與 `ws://127.0.0.1:8080/ws` 連線
* 傳送Message: `{"command": "/hello", "data":"hi, it is a websocket test" }`