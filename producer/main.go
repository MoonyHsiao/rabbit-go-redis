package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
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
		log.Fatalln("Create wait Queue fail:", err)
	}
	err = mqCh.ExchangeDeclare(constant.WaitExchange, amqp.ExchangeDirect, true, false, false, false, nil)

	if err != nil {
		log.Fatalln("Create wait Exchange fail:", err)
	}
	err = mqCh.QueueBind(constant.WaitQueue, constant.WaitRoutingKey, constant.WaitExchange, false, nil)

	if err != nil {
		log.Fatalln("Wait:Queue、Exchange、routing-key Bind fail:", err)
	}
	_, err = mqCh.QueueDeclare(constant.NormalQueue, true, false, false, false, nil)

	if err != nil {
		log.Fatalln("Create Normal Queue fail:", err)
	}
	err = mqCh.ExchangeDeclare(constant.NormalExchange, amqp.ExchangeDirect, true, false, false, false, nil)

	if err != nil {
		log.Fatalln("Create Normal Exchange fail:", err)
	}
	err = mqCh.QueueBind(constant.NormalQueue, constant.NormalRoutingKey, constant.NormalExchange, false, nil)

	if err != nil {
		log.Fatalln("Normal:Queue、Exchange、routing-key Bind fail:", err)
	}

	ctx := context.Background()
	rdb = util.NewRedisClient(ctx)
	count = 0
	port := viper.GetString("bot.api_port")
	router := gin.Default()
	v1 := router.Group(viper.GetString("bot.api_baseurl"))
	{
		v1.POST("/NewOrder", CreateNewOrder)
		v1.POST("/MultNewOrder", CreateMultNewOrder)
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

func CreateMultNewOrder(ctx *gin.Context) {

	var wg sync.WaitGroup

	list := []int{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := callanother()
			list = append(list, status)
		}()
	}

	wg.Wait()

	ctx.JSON(http.StatusOK, gin.H{
		"Status":   "SUCCESS",
		"SendData": list,
	})

}

func callanother() int {

	url := "http://127.0.0.1:18086/api/NewOrder"
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return 400
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return 400
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return 400
	}

	fmt.Println(string(body))

	return res.StatusCode
}
