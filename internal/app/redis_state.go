package app

import (
	"context"

	"github.com/ernestoponce27/db-tui/internal/db"
	"github.com/ernestoponce27/db-tui/internal/redis"
)

// redisState owns Redis-only connection and navigator state.
type redisState struct {
	client          *redis.Client
	connect         RedisConnectFunc
	database        int
	databases       []int
	pages           []db.RowPage
	cursor          redis.ScanCursor
	pageIndex       int
	highlighted     int
	navigatorOffset int
	request         uint64
	loadCancel      context.CancelFunc
}

func (r *redisState) reset() {
	connect := r.connect
	if r.loadCancel != nil {
		r.loadCancel()
	}
	*r = redisState{connect: connect}
}
