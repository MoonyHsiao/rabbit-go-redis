# rabbit-go-redis

The purpose of the project is to delay the processing of requests that cannot be handled immediately (e.g. requests from a previous user that are still being processed)
So, RabbitMQ will retry sending the message after 30 seconds 
Messages sent out by RabbitMQ will be processed in parallel by consumers to avoid duplication. 
A distributed lock from Redis is used to achieve this.


The content used is a distributed lock from Redis + dead letter message from RabbitMQ.

# 參考資料
https://medium.com/@lalayueh/%E5%A6%82%E4%BD%95%E5%84%AA%E9%9B%85%E5%9C%B0%E5%9C%A8rabbitmq%E5%AF%A6%E7%8F%BE%E5%A4%B1%E6%95%97%E9%87%8D%E8%A9%A6-c050efd72cdb


curl --location --request POST 'http://127.0.0.1:18086/api/NewOrder'