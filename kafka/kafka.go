/*
Copyright 2016 caicloud authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kafka

import (
	"fmt"
	"sync"
	"time"

	"github.com/caicloud/cyclone/pkg/log"
	"github.com/optiopay/kafka"
	"github.com/optiopay/kafka/proto"
)

var (
	broker   *kafka.Broker
	producer kafka.Producer
	// boolean value for connect to kafka server status
	bConnected = false
	// ErrNoData is the err type for no data.
	ErrNoData           = kafka.ErrNoData
	lockFileWatchSwitch sync.RWMutex
	lockReproduce       sync.RWMutex
	lockConsumer        sync.RWMutex
	kafkaAddrs          []string
)

const (
	Partition          = 0
	ConsumeRetryLimit  = 5
	Broker             = "log-client"
	reproduceMaxTimes  = 3
	reconsumerMaxTimes = 3

	// retryInterval represents the interval time to retry the connection to Kafka if it is offline.
	retryInterval = 1 * time.Second
)

// Dial dial to kafka server
func Dial(sKafkaAddrs []string) (err error) {
	kafkaAddrs = sKafkaAddrs
	conf := kafka.NewBrokerConf(Broker)
	conf.AllowTopicCreation = true
	broker, err = kafka.Dial(sKafkaAddrs, conf)
	if nil != err {
		bConnected = false
		return err
	}

	bConnected = true
	producer = broker.Producer(kafka.NewProducerConf())
	return nil
}

// Close close the link to kafka server
func Close() {
	bConnected = false
	broker.Close()
}

// redial redial to kafka server
func redial() {
	log.Infof("redial kafka!")
	Close()
	Dial(kafkaAddrs)
}

// IsConnected get the status of the link to kafka server
func IsConnected() bool {
	return bConnected
}

// Produce produce the message to the special topic
func Produce(sTopic string, byrMsg []byte) (err error) {
	lockReproduce.Lock()
	defer lockReproduce.Unlock()

	return produce(sTopic, byrMsg)
}

// produce produce the message to the special topic. If kafka client offline, it
// will redial kafka server and produce message again. If produce message to kafka
// server 3 times continuously, the function will exit and return error.
func produce(sTopic string, byrMsg []byte) (err error) {
	msg := &proto.Message{Value: byrMsg}
	_, err = producer.Produce(sTopic, Partition, msg)
	for retryTimes := 0; err != nil && retryTimes < reproduceMaxTimes; retryTimes++ {
		log.Errorf("Can't produce message to %s:%d: %s", sTopic, Partition, err.Error())

		// Sleep and retry later.
		time.Sleep(retryInterval)
		redial()
		_, err = producer.Produce(sTopic, Partition, msg)
	}

	return err
}

// NewConsumer create a new cosumer to the special topic
func NewConsumer(sTopic string) (kafka.Consumer, error) {
	lockConsumer.Lock()
	defer lockConsumer.Unlock()

	return newConsumer(sTopic)
}

// newConsumer create a new consumer to the special topic. If kafka client offline,
// it will redial kafka server and create new consumer again. If create consumer
// 3 times continuously, the function will exit and return error.
func newConsumer(sTopic string) (kafka.Consumer, error) {
	conf := kafka.NewConsumerConf(sTopic, Partition)
	conf.StartOffset = kafka.StartOffsetOldest
	conf.RetryLimit = ConsumeRetryLimit
	consumer, err := broker.Consumer(conf)
	for retryTimes := 0; err != nil && retryTimes < reconsumerMaxTimes; retryTimes++ {
		log.Errorf("Can't create kafka consumer for %s:%d: %s", sTopic, Partition, err.Error())

		// Sleep and retry later.
		time.Sleep(retryInterval)
		redial()
		consumer, err = broker.Consumer(conf)
	}

	if err == nil {
		return consumer, nil
	}

	return nil, fmt.Errorf("Fail to new consumer after %d retries as %s", reconsumerMaxTimes, err.Error())
}
