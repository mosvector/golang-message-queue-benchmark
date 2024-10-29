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
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("server called")
		runServer()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runServer() {
	cfg := new(tls.Config)
	cfg.RootCAs = x509.NewCertPool()
	cfg.InsecureSkipVerify = true

	cfg.ServerName = "mq.example.com"
	if ca, err := os.ReadFile("cert/ca.pem"); err == nil {

		cfg.RootCAs.AppendCertsFromPEM(ca)
	}

	if cert, err := tls.LoadX509KeyPair("cert/rabbitmq-client-dev.pem", "cert/rabbitmq-client-dev-key.pem"); err == nil {
		log.Println("Add client cert")
		cfg.Certificates = append(cfg.Certificates, cert)
	}

	conn, _ := amqp.DialTLS_ExternalAuth("amqps://192.168.10.251:38600/", cfg)
	exchange := "platform.api"
	exchange = ""
	defer conn.Close()

	go func(con *amqp.Connection) {
		channel, _ := conn.Channel()
		defer channel.Close()
		durable, exclusive := false, false
		autoDelete, noWait := false, true

		//_ = channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
		cq, _ := channel.QueueDeclare("command", durable, autoDelete, exclusive, noWait, nil)
		log.Println("Create queue")
		//channel.QueueBind(cq.Name, "command", exchange, false, nil)
		//channel.QueueBind(cq.Name, "*.command", exchange, false, nil)
		autoAck, exclusive, noLocal, noWait := false, false, false, false
		messages, _ := channel.Consume(cq.Name, "", autoAck, exclusive, noLocal, noWait, nil)
		multiAck := false
		for msg := range messages {
			fmt.Println("Body:", string(msg.Body), "Timestamp:", msg.Timestamp)
			msg.Ack(multiAck)

			var packet util.Packet
			json.Unmarshal(msg.Body, &packet)
			packet.CompletedTime = strconv.FormatInt(time.Now().UnixNano(), 10)

			data, _ := json.Marshal(&packet)

			reply := amqp.Publishing{
				DeliveryMode:    1,
				Timestamp:       time.Now(),
				ContentType:     "application/json",
				Body:            data,
				CorrelationId:   msg.CorrelationId,
				Type:            "order.response",
				ContentEncoding: "plain",
			}
			mandatory, immediate := false, false

			log.Println("CorrelationId", msg.CorrelationId)
			log.Println("Reply", msg.ReplyTo)
			channel.Publish(exchange, msg.ReplyTo, mandatory, immediate, reply)
		}
	}(conn)

	select {}
}
