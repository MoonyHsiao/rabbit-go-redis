package client

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/MoonyHsiao/rabbit-go-redis/constant"
	"github.com/MoonyHsiao/rabbit-go-redis/util"
	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Manager struct {
	broadcast  chan []byte
	topicState map[string]string
	rdb        *redis.Client
	mq         *util.RabbitMQ
	ctx        context.Context
}

func NewManager(rdb *redis.Client, ctx context.Context, mq *util.RabbitMQ) *Manager {
	return &Manager{
		broadcast:  make(chan []byte),
		topicState: make(map[string]string),
		mq:         mq,
		rdb:        rdb,
		ctx:        ctx,
	}
}

func (mg *Manager) FailConsumer() {

	defer func() {
		mg.mq.Close()
		log.Printf("defer close manager\n")
	}()
	ch := mg.mq.Channel

	ch.ExchangeDeclare("main_exchange",
		"direct",
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)

	q, err := ch.QueueDeclare(
		"main_queue", // name
		false,        // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	// amqp.Table{"x-dead-letter-exchange": "hello"}, // arguments
	if err != nil {
		log.Fatalln("Failed to declare a queue:", err)
	}

	ch.QueueBind("main_queue",
		"main_routing_key",
		"main_exchange",
		false,
		nil)

	_, err2 := ch.QueueDeclare(
		"delay_queue", // name
		false,         // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "main_exchange",
			"x-message-ttl":             15000,
			"x-dead-letter-routing-key": "main_routing_key"},
	)

	if err2 != nil {
		log.Fatalln("Failed to declare a queue:", err2)
	}

	for {
		msgs, ok, err := ch.Get(q.Name, false)
		// fmt.Printf("%v,%v", msgs, ok)
		if err != nil {
			log.Fatalln("Failed to register a consumer:", err)
		}
		time.Sleep(2 * time.Second)

		if ok {

			log.Printf("dealMessage: %s", msgs.Body)

			msgs.Nack(false, false)

		}

	}
}

func (mg *Manager) StartConsumerWithRedis() {

	defer func() {
		mg.mq.Close()
		log.Printf("defer close manager\n")
	}()
	ch := mg.mq.Channel

	msgs, err := ch.Consume(
		constant.NormalQueue, // queue
		"",                   // consumer
		false,                // auto-ack
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)

	if err != nil {
		log.Fatalln("Failed to register a consumer:", err)
	}

	for d := range msgs {
		go mg.dealMessage(d)
		d.Ack(false)
	}

	// for d := range msgs {
	// 	lockV := util.GetRandValue()
	// 	if lockV%2 == 0 {
	// 		log.Printf("接收的消息未處理完成再次發送: %s", d.Body)

	// 		err = ch.Publish(constant.WaitExchange, constant.WaitRoutingKey, false, false, amqp.Publishing{
	// 			ContentType: "text/plain",
	// 			Body:        []byte(d.Body),
	// 		})
	// 	} else {
	// 		log.Printf("接收的消息處理完畢: %s", d.Body)
	// 	}
	// 	d.Ack(false)
	// }

}

func (mg *Manager) dealMessage(d amqp.Delivery) {

	myString := string(d.Body)
	myStringArray := strings.Split(myString, "-")
	lockK := fmt.Sprintf("%s-lotto", myStringArray[0])
	lockV, err := strconv.Atoi(myStringArray[1])

	ch := mg.mq.Channel

	rdb := mg.rdb
	ctx := mg.ctx
	set, err := rdb.SetNX(ctx, lockK, lockV, constant.Expiration).Result()

	if err != nil || set == false {
		log.Printf("接收的消息未處理完成再次發送: %s,%v", d.Body, set)
		err = ch.Publish(constant.WaitExchange, constant.WaitRoutingKey, false, false, amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(d.Body),
		})
		return
	}
	log.Printf("redis lock success lockK:%v,dealing data :%s\n", lockK, d.Body)
	sec := util.GetRandDuration()
	// do what you wanted and spend times
	time.Sleep(sec)

	val := redisDelByKeyWhenValueEquals(ctx, rdb, lockK, lockV)
	log.Printf("接收的消息處理完畢: %s,%v 處理時間:%v", d.Body, val, sec)
}

func redisDelByKeyWhenValueEquals(ctx context.Context, rdb *redis.Client, key string, value interface{}) bool {
	lua := `
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
else
	return 0
end
`
	scriptKeys := []string{key}

	val, err := rdb.Eval(ctx, lua, scriptKeys, value).Result()
	if err != nil {
		fmt.Println("解鎖失敗")
	}

	return val == int64(1)
}
