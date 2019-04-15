package ratelimit

import (
	"math"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

const Inf = math.MaxFloat64

type Limiter struct {
	redis *redis.Client
	limit int64
	burst int64
}

func NewLimiter(redis *redis.Client, rate, burst int64) *Limiter {
	return &Limiter{
		redis: redis,
		limit: rate,
		burst: burst,
	}
}

func (l *Limiter) ReserveN(name string) (count int64, delay time.Duration, allow bool) {
	ltime := time.Now().Unix()
	lmin := float64(ltime - 1)
	l.redis.ZRemRangeByScore(name, strconv.FormatFloat(-1*Inf, 'f', 6, 64), strconv.FormatFloat(lmin, 'f', 6, 64))

	llast := l.redis.ZRange(name, -1, -1)
	lnext := ltime

	for _, v := range llast.Val() {
		lprev, _ := strconv.ParseInt(v, 10, 64)
		lnext = lprev + 1/l.limit
		break
	}

	if ltime > lnext {
		lnext = ltime
	}

	count = l.redis.ZCard(name).Val()
	allow = count <= l.burst

	if allow {
		lmember := redis.Z{Score: float64(lnext), Member: lnext}
		l.redis.ZAdd(name, lmember)
	}

	return count, time.Duration(lnext - ltime), allow
}
