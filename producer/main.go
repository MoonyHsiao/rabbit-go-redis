package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MoonyHsiao/rabbit-go-redis/constant"
	"github.com/MoonyHsiao/rabbit-go-redis/util"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

var count int
var rdb *redis.Client
var mq *util.RabbitMQ

var expiration time.Duration

func main() {

	// var wg sync.WaitGroup

	// list := []int{}
	// for i := 0; i < 5; i++ {
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		status := callanother()
	// 		list = append(list, status)
	// 	}()
	// }

	// wg.Wait()

	// fmt.Printf("status:%v\n", list)
	// # ========== 1.创建连接 ==========
	util.InitViper()
	mq = util.NewRabbitMQ()
	defer mq.Close()
	mqCh := mq.Channel
	var err error
	_, err = mqCh.QueueDeclare(constant.WaitQueue, true, false, false, false, amqp.Table{
		"x-message-ttl":             15000,                     // 消息過期時間,毫秒
		"x-dead-letter-exchange":    constant.NormalExchange,   // 指定死信交換機
		"x-dead-letter-routing-key": constant.NormalRoutingKey, // 指定死信routing-key
	})
	if err != nil {
		log.Fatalln("創建wait隊列失敗:", err)
	}
	// 聲明交換機
	err = mqCh.ExchangeDeclare(constant.WaitExchange, amqp.ExchangeDirect, true, false, false, false, nil)

	if err != nil {
		log.Fatalln("創建wait交換機失敗:", err)
	}
	// 隊列綁定（將隊列、routing-key、交換機三者綁定到一起）
	err = mqCh.QueueBind(constant.WaitQueue, constant.WaitRoutingKey, constant.WaitExchange, false, nil)

	if err != nil {
		log.Fatalln("Wait:队列、交换机、routing-key 绑定失败", err)
	}
	// # ========== 3.設置死信隊列（隊列、交換機、綁定） ==========
	// 聲明死信隊列
	// args 為 nil。切記不要給死信隊列設置消息過期時間,否則失效的消息進入死信隊列後會再次過期。
	_, err = mqCh.QueueDeclare(constant.NormalQueue, true, false, false, false, nil)

	if err != nil {
		log.Fatalln("創建Normal隊列失敗:", err)
	}
	// 聲明交換機
	err = mqCh.ExchangeDeclare(constant.NormalExchange, amqp.ExchangeDirect, true, false, false, false, nil)

	if err != nil {
		log.Fatalln("創建Normal隊列失敗:", err)
	}
	// 隊列綁定（將隊列、routing-key、交換機三者綁定到一起）
	err = mqCh.QueueBind(constant.NormalQueue, constant.NormalRoutingKey, constant.NormalExchange, false, nil)

	if err != nil {
		log.Fatalln("Normal:隊列、交換機、routing-key 綁定失敗", err)
	}

	ctx := context.Background()
	rdb = util.NewRedisClient(ctx)
	count = 0
	port := viper.GetString("bot.api_port")
	router := gin.Default()
	v1 := router.Group(viper.GetString("bot.api_baseurl"))
	{
		v1.POST("/NewOrder", CreateNewOrder)
	}
	log.Println("http server started on" + port)
	router.Run(port)
}

func CreateNewOrder(ctx *gin.Context) {

	user := util.GetRandUser()
	message := fmt.Sprintf("%s-%v", user, count)
	// fmt.Println(message)
	mqCh := mq.Channel
	err := mqCh.Publish(constant.WaitExchange, constant.WaitRoutingKey, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message),
	})

	if err != nil {
		log.Println("發布消息失敗:", err)
	}
	count++
	ctx.JSON(http.StatusOK, gin.H{
		"Status": "SUCCESS",
	})

}

func welcome_fail_back(ctx *gin.Context) {

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	util.FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	util.FailOnError(err, "Failed to open a channel")
	defer ch.Close()

	ch.ExchangeDeclare("main_exchange",
		"direct",
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)

	// amqp.Table{"x-dead-letter-exchange": "user_dlx", "x-message-ttl": time.Second * 5}, // arguments
	util.FailOnError(err, "Failed to declare a queue")
	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body := fmt.Sprintf("%s: %v", "Hello World!", count)
	// user_dlx
	err = ch.PublishWithContext(ctx2,
		"main_exchange",    // exchange
		"main_routing_key", // routing key
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	util.FailOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s\n", body)
	count++
	ctx.JSON(http.StatusOK, gin.H{
		"Status": "SUCCESS",
	})

}
