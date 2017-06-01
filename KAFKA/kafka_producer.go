package KAFKA

import (
	"fmt"
	"github.com/Shopify/sarama"
	_ "os"
	"strings"
	"sync"
	_ "syscall"
)

var Signals chan string

func InitAsyncProducer(aressList string, topicName string) (sarama.AsyncProducer, error) {
	configkafka := sarama.NewConfig()
	fmt.Println(configkafka)
	configkafka.Producer.Return.Successes = true
	//设置默认发送方式为RoundRobin
	configkafka.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	producer, err := sarama.NewAsyncProducer(strings.Split(aressList, ","), configkafka)
	if err != nil {
		fmt.Println("NewAsyncProducer ERR:", err)
		return nil, err
	}

	Signals = make(chan string, 1)
	//signal.Notify(Signals, syscall.SIGHUP)

	go expectResults(producer)

	return producer, nil
}

func expectResults(p sarama.AsyncProducer) {
	for {
		select {
		case msg := <-p.Errors():
			if msg.Err != nil {
				fmt.Println("Message send error")
			}

		case msg := <-p.Successes():
			if msg == nil {
				fmt.Println("Message send error..")
			}
		case <-Signals:
			fmt.Println("recieve the exit signal,now exit... ")
			goto ProducerLoop
		}

	}

ProducerLoop:
}

func SendMsg(producer sarama.AsyncProducer, topicName, sendmsg string) {
	select {
	//发送消息并异步捕获异常
	case producer.Input() <- &sarama.ProducerMessage{Topic: topicName, Key: nil, Value: sarama.StringEncoder(sendmsg)}:
	default:
		return
	}

}

func CloseProducer(producer sarama.AsyncProducer) {
	var wg sync.WaitGroup
	producer.AsyncClose()
	Signals <- "exit"
	wg.Add(2)
	go func() {
		for range producer.Successes() {
			fmt.Println("Unexpected message on Successes()")
		}
		wg.Done()
	}()
	go func() {
		for msg := range producer.Errors() {
			fmt.Println(msg.Err)
		}
		wg.Done()
	}()
	wg.Wait()

}
