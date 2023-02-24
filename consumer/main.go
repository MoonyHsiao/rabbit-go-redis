package main

import (
	"context"
	"log"

	"github.com/MoonyHsiao/rabbit-go-redis/client"
	"github.com/MoonyHsiao/rabbit-go-redis/util"
)

func main() {
	util.InitViper()
	ctx := context.Background()
	rdb := util.NewRedisClient(ctx)
	mqClient := util.NewRabbitMQ()

	manager := client.NewManager(rdb, ctx, mqClient)

	go manager.StartConsumerWithRedis()

	var forever chan struct{}
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
