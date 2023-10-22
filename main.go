package main

import (
	"context"
	"fmt"
	"redis-id-generator/pkg/idgen"
	"sync"
	"time"
)

var wg sync.WaitGroup

func main() {
	store := idgen.NewRedisIdStore()
	// step > qpm
	idgen := idgen.NewIdGenrator(store, idgen.With2BytesRandomFilter())
	now := time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 100; j++ {
				id, err := idgen.GetId(context.Background(), "test")
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println(id)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Printf("%s\n", time.Since(now))
}
