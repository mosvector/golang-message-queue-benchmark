/*
Copyright © 2022 Raymond Lin <raymondlin@live.hk>
*/
package util

import (
	"math/rand"
	"time"
)

type Packet struct {
	ClientAccId   string `json:"client_acc_id"`
	Action        string `json:"action"`
	IssueTime     string `json:"issue_time"`
	CompletedTime string `json:"completed_time"`
}

type AddOrderAction struct {
	Type          string `json:"type"`
	MsgNum        string `json:"msgnum"`
	BsFlag        string `json:"bs_flag"`
	ClientAccCode string `json:"client_acc_code"`
	ExchangeCode  string `json:"exchange_code"`
	ProductCode   string `json:"product_code"`
	OrderType     string `json:"order_type"`
	Price         string `json:"price"`
	Qty           string `json:"qty"`
	Reference     string `json:"reference"`
}

func Unix() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func RandomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
