package generator

import (
	"time"

	"github.com/allan-deng/redis-id-generator/pkg/idgen"

	log "github.com/sirupsen/logrus"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

var IdGen *idgen.IdGenerator

func IdGenInit() {
	addr := viper.GetString("redis.addr")
	pwd := viper.GetString("redis.password")

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       0,
	})
	// check redis connect status...
	_, err := client.Ping(client.Context()).Result()
	if err != nil {
		log.Fatalf("canot connect to redis: %v.", addr)
	}

	log.Infof("connect to redis %v succ", addr)

	store := idgen.NewRedisIdStore(client)

	opts := make([]idgen.Option, 0)
	opts = append(opts, idgen.With2BytesRandomFilter())

	if retry_times := viper.GetInt("idgen.preload_retry_times"); retry_times > 0 {
		opts = append(opts, idgen.WithPreloadRetryTimes(retry_times))
	}
	if step := viper.GetInt64("idgen.default_step"); step > 0 {
		opts = append(opts, idgen.WithStep(step))
	}
	if timeout := viper.GetDuration("idgen.preload_timeout") * time.Millisecond; timeout > 0 {
		opts = append(opts, idgen.WithPreloadTimeout(timeout))
	}
	if expire := viper.GetDuration("idgen.biztag_expire_time")* time.Second; expire > 0 {
		opts = append(opts, idgen.WithExpireTime(expire))
	}

	IdGen = idgen.NewIdGenrator(store, opts...)
}
