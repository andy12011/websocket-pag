package ws

type NewParams struct {
	IsCrossDomain bool
	SendTimeout   int
	BroadcastBuf  int
	PoolBuf       int
	PoolWorkers   int
}

func New(params NewParams) WSBrokerInterface {
	if params.SendTimeout == 0 {
		params.SendTimeout = 10
	}
	if params.BroadcastBuf == 0 {
		params.BroadcastBuf = 1000
	}
	if params.PoolBuf == 0 {
		params.PoolBuf = 10000
	}
	if params.PoolWorkers == 0 {
		params.PoolWorkers = 10
	}
	return newWSBroker(params)
}

func Default() WSBrokerInterface {
	return newWSBroker(NewParams{
		IsCrossDomain: true,
		SendTimeout:   10,
		BroadcastBuf:  1000,
		PoolBuf:       10000,
		PoolWorkers:   10,
	})
}
