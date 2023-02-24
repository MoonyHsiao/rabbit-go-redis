package util

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

var UserNameMap = map[int]string{
	0: "Eric",
	1: "John",
	2: "Alxe",
	3: "Ray",
}

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewRabbitMQ() *RabbitMQ {
	MqUrl := viper.GetString("rabbitmq.url")
	conn, err := amqp.Dial(MqUrl)
	FailOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	FailOnError(err, "Failed to open a channel")

	return &RabbitMQ{
		Conn:    conn,
		Channel: ch,
	}
}

func (r RabbitMQ) Close() {
	r.Channel.Close()
	r.Conn.Close()
}

func FailOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func GetRandValue() int {
	rand.Seed(time.Now().UnixNano())
	min := 0
	max := 4
	return rand.Intn(max-min) + min
}

func GetRandUser() string {
	rand.Seed(time.Now().UnixNano())
	min := 0
	max := len(UserNameMap) - 1
	res := rand.Intn(max-min) + min
	return UserNameMap[res]
}

func GetRandDuration() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := 2
	max := 5
	return time.Duration(rand.Intn(max-min)+min) * time.Second
}

func NewRedisClient(ctx context.Context) *redis.Client {
	client := redis.NewClient(redisArgs())

	pong, err := client.Ping(ctx).Result()
	FailOnError(err, "Failed to open a channel")

	log.Println(pong)
	return client
}

func redisArgs() *redis.Options {
	redis_url := viper.GetString("redis.url")
	redis_password := viper.GetString("redis.password")
	redis_db := viper.GetInt("redis.db")
	redis_poolsize := viper.GetInt("redis.poolsize")
	return &redis.Options{Addr: redis_url,
		Password: redis_password,
		DB:       redis_db,
		PoolSize: redis_poolsize,
	}
}

func SetDefault() {
	viper.SetDefault("bot.test_default_value", 10)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.poolsize", 5)
}

func InitViper() {

	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	SetDefault()
	viper.AddConfigPath("./config")
	viper.SetConfigName("setting")

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		log.Printf("Using ENV\n")
	}

	log.Println("redis.url = " + viper.GetString("redis_url"))
	// log.Println("bot.test_default_value = " + viper.GetString("bot.test_default_value"))
}
