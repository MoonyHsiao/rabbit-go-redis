package constant

import "time"

const WaitQueue = "wait-queue"
const WaitExchange = "wait-exchange"
const WaitRoutingKey = "wait-key"
const NormalQueue = "normal-queue"
const NormalExchange = "normal-exchange"
const NormalRoutingKey = "normal-key"
const Expiration = time.Second * 10
