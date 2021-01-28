# ws

新的 websocket 套件

## Installation
取得專案
```shell
$ go get -i gitlab.paradise-soft.com/glob/ws
```

匯入到你的專案中
```go
import "gitlab.paradise-soft.com/glob/ws"
```

## Quick start

```go
package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.paradise-soft.com/glob/ws"
)

var (
	ginEngine *gin.Engine
	wsEngine  ws.WSBrokerInterface
)

func main() {
	ginEngine = gin.Default()
	wsEngine = ws.Default()

	ginEngine.GET("/ws", upgradeHttp)

	ginEngine.Run(":8080")
}

func upgradeHttp(c *gin.Context) {
	ctx := setPlayerID(c)
	wsEngine.UpgradeHttp(ctx, c.Writer, c.Request)
}

func setPlayerID(ctx context.Context) context.Context {
	return context.WithValue(ctx, "playerID", "0d0d85d2-4e12-41c1-b34e-f8143e347d61")
}
```