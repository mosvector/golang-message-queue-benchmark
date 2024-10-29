/*
Copyright © 2022 Raymond Lin <raymondlin@live.hk>
*/
package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"strconv"
	"time"

	"github.com/golang-message-queue-benchmark/pkg/util"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("client called")
		t, _ := cmd.Flags().GetInt("interval")
		userId := args[0]
		log.Println("userId", userId, "interval", t)
		runClient(userId, t)
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clientCmd.PersistentFlags().String("foo", "", "A help for foo")
	clientCmd.PersistentFlags().IntP("interval", "t", 1, "interval")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clientCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runClient(userId string, interval int) {
	cfg := new(tls.Config)
	cfg.RootCAs = x509.NewCertPool()
	cfg.InsecureSkipVerify = true

	cfg.ServerName = "mq.example.com"
	if ca, err := os.ReadFile("cert/ca.pem"); err == nil {
		cfg.RootCAs.AppendCertsFromPEM(ca)
	}

	if cert, err := tls.LoadX509KeyPair("cert/rabbitmq-client-dev.pem", "cert/rabbitmq-client-dev-key.pem"); err == nil {
		fmt.Println("Add client cert")
		cfg.Certificates = append(cfg.Certificates, cert)
	}

	conn, _ := amqp.DialTLS_ExternalAuth("amqps://192.168.10.251:38600/", cfg)
	exchange := "platform.api"
	exchange = ""

	defer conn.Close()

	name := make(chan string)

	go func(con *amqp.Connection) {
		channel, _ := conn.Channel()
		defer channel.Close()
		durable, exclusive := false, true
		autoDelete, noWait := false, false
		//_ = channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
		rq, _ := channel.QueueDeclare("", durable, autoDelete, exclusive, noWait, nil)
		log.Println(rq.Name)
		name <- rq.Name

		_ = channel.Qos(1, 0, false)
		//channel.QueueBind(rq.Name, rq.Name, exchange, false, nil)
		//channel.QueueBind(rq.Name, "*.result", exchange, false, nil)
		autoAck, exclusive, noLocal, noWait := false, false, false, false
		messages, _ := channel.Consume(rq.Name, "", autoAck, exclusive, noLocal, noWait, nil)
		multiAck := false
		for msg := range messages {
			log.Println("Recv message")
			var packet util.Packet
			json.Unmarshal(msg.Body, &packet)

			issueTime, _ := strconv.Atoi(packet.IssueTime)
			completedTime, _ := strconv.Atoi(packet.CompletedTime)
			diff := int64(completedTime-issueTime) / int64(time.Millisecond) * 1000

			fmt.Println("Body:", string(msg.Body), "Diff:", diff, "ns")
			msg.Ack(multiAck)
		}

		IfUnused, IfEmpty, noWait := false, false, false
		i, _ := channel.QueueDelete(rq.Name, IfUnused, IfEmpty, noWait)
		log.Println("Count", i)

	}(conn)

	go func(con *amqp.Connection) {
		timer := time.NewTicker(time.Duration(interval) * time.Millisecond)
		channel, _ := conn.Channel()
		defer channel.Close()

		queueName := <-name
		log.Println("Set queueName", queueName)
		for t := range timer.C {
			data, _ := json.Marshal(util.Packet{
				ClientAccId: userId,
				Action:      "place_order",
				IssueTime:   strconv.FormatInt(time.Now().UnixNano(), 10),
			})

			corrId := uuid.New().String()
			log.Println("Queue", queueName, "CorrId", corrId)

			msg := amqp.Publishing{
				DeliveryMode:    1,
				Timestamp:       t,
				ContentType:     "application/json",
				CorrelationId:   corrId,
				ReplyTo:         queueName,
				Body:            data,
				Type:            "order.request",
				ContentEncoding: "plain",
				MessageId:       "1",
				Headers:         amqp.Table{"token": "abcd"},
			}
			mandatory, immediate := false, false
			log.Println("Send request to", exchange, "command")
			channel.Publish(exchange, "command", mandatory, immediate, msg)
		}
	}(conn)

	select {}
}
