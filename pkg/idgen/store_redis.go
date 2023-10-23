package idgen

import (
	"context"

	"github.com/go-redis/redis/v8"
)

var getSegScript = `
local key = KEYS[1]
local step = ARGV[1]

local exists = redis.call("EXISTS", key)

if exists == 1 then
	local max = redis.call("HGET", key, "max")
	local currentStep = redis.call("HGET", key, "step")
	local newMax = tonumber(max) + tonumber(currentStep)
	redis.call("HSET", key, "max", newMax)
	return newMax
else
	redis.call("HSET", key, "step", step)
	redis.call("HSET", key, "max", step)
	return step
end
`

var getSegLua = redis.NewScript(getSegScript)

type RedisIdStore struct {
	redisClient *redis.Client
}

func NewRedisIdStore(client *redis.Client) *RedisIdStore {
	return &RedisIdStore{
		redisClient: client,
	}
}

func (this *RedisIdStore) GetNextSegment(ctx context.Context, bizTag string, step int64) (*Seg, error) {
	// If there is no key in redis, it is created; otherwise, it is updated
	newMaxId, err := this.getNextMaxId(ctx, bizTag, step)
	if err != nil {
		return nil, err
	}
	return &Seg{
		BizTag: bizTag,
		MaxId:  newMaxId,
		Step:   step,
	}, nil
}

func (this *RedisIdStore) genRedisKey(bizTag string) string {
	return "idgen:" + bizTag
}

func (this *RedisIdStore) getNextMaxId(ctx context.Context, bizTag string, step int64) (int64, error) {
	keys := []string{this.genRedisKey(bizTag)}
	res, err := getSegLua.Run(ctx, this.redisClient, keys, step).Int64()
	return res, err
}
