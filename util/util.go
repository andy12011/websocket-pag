package util

import (
	"context"
	"log"
	"time"
)

func TimeTaken(ctx context.Context, t time.Time, name string) {
	chainID := ctx.Value("ChainID").(string)
	elapsed := time.Since(t)
	log.Printf("[%s] %s took %s\n", chainID, name, elapsed)
}
