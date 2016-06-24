package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/koding/asgd"
	"github.com/koding/logging"
)

const Name = "asgd"

func main() {
	conf, err := asgd.Configure()
	if err != nil {
		log.Fatal("Reading config failed: ", err.Error())
	}

	// system name defines all resource names
	systemName := fmt.Sprintf("%s-%s", "asgd", conf.Name)

	log := logging.NewCustom(Name, conf.Debug)
	// remove formatting from call stack and output correct line
	log.SetCallDepth(1)

	// create lifecycle
	l := asgd.NewLifeCycle(conf.Session, log, conf.AutoScalingName)

	// configure lifecycle with system name
	if err := l.Configure(systemName); err != nil {
		log.Fatal(err.Error())
	}

	done := registerSignalHandler(l, log)

	// listen to lifecycle events
	if err := l.Listen(process); err != nil {
		log.Fatal(err.Error())
	}

	<-done
}

func process(instances []*ec2.Instance) error {
	return nil
}

func registerSignalHandler(l *asgd.LifeCycle, log logging.Logger) chan struct{} {
	done := make(chan struct{}, 1)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals)

		signal := <-signals
		switch signal {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP, syscall.SIGKILL:
			log.Info("recieved exit signal, closing...")
			err := l.Close()
			if err != nil {
				log.Critical(err.Error())
			}
			close(done)
		}

	}()
	return done
}
