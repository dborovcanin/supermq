// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats_test

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	broker "github.com/nats-io/nats.go"
	dockertest "github.com/ory/dockertest/v3"
)

const (
	version = "1.3.0"
	port    = "4222/tcp"
)

var (
	publisher messaging.Publisher
	pubsub    messaging.PubSub
	conn      *broker.Conn
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.Run("nats", version, []string{})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}
	handleInterrupt(pool, container)

	logger, err := logger.New(os.Stdout, "error")
	address := fmt.Sprintf("%s:%s", "localhost", container.GetPort(port))

	if err := pool.Retry(func() error {
		conn, err = broker.Connect(address)
		return err
	}); err != nil {
		log.Fatalf("Could not connect client to docker: %s", err)
	}

	if err := pool.Retry(func() error {
		publisher, err = nats.NewPublisher(address)
		return err
	}); err != nil {
		log.Fatalf("Could not connect publisher to docker: %s", err)
	}

	if err != nil {
		log.Fatalf(err.Error())
	}
	if err := pool.Retry(func() error {
		pubsub, err = nats.NewPubSub(address, "", logger)
		return err
	}); err != nil {
		log.Fatalf("Could not connect pubsub to docker: %s", err)
	}

	code := m.Run()
	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}

func handleInterrupt(pool *dockertest.Pool, container *dockertest.Resource) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		if err := pool.Purge(container); err != nil {
			log.Fatalf("Could not purge container: %s", err)
		}
		os.Exit(0)
	}()
}
