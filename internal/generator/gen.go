package generator

import (
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
	IdGen = idgen.NewIdGenrator(store, idgen.With2BytesRandomFilter())
}
