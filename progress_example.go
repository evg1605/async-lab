package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/evg1605/async-lab/last_progress"
)

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	ctx, cancel := context.WithCancel(context.Background())
	pr := last_progress.NewProgress()

	log.Println("start server")
	go startServer(ctx, wg, pr, time.Second)

	log.Println("start client")
	go startClient(ctx, wg, pr, time.Second*3)

	time.Sleep(time.Second * 50)

	log.Println("stop server and client")
	cancel()
	wg.Wait()

	log.Println("all done")
}

func startServer(ctx context.Context, wg *sync.WaitGroup, pr last_progress.Producer, countJob time.Duration) {
	defer wg.Done()

	for i := 0; ctx.Err() == nil; i++ {
		v := fmt.Sprintf("item-%v", i)
		pr.SetLastProgress(v)
		log.Printf("set progress %s", v)
		time.Sleep(countJob)
	}
}

func startClient(ctx context.Context, wg *sync.WaitGroup, pr last_progress.Consumer, countJob time.Duration) {
	defer wg.Done()

	for {
		select {
		case v := <-pr.GetLastProgress():
			log.Printf("received progress %s", v)
			time.Sleep(countJob)
		case <-ctx.Done():
			return
		}
	}
}
