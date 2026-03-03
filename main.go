package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/security/crypto"
	log "github.com/sirupsen/logrus"
)

type Executor struct {
	coloniesServerHost string
	coloniesServerPort int
	coloniesInsecure   bool
	colonyPrvKey       string
	colonyID           string
	colonyName         string
	executorID         string
	executorPrvKey     string
	executorName       string
	executorType       string
	ctx                context.Context
	cancel             context.CancelFunc
	client             *client.ColoniesClient
}

type ExecutorOption func(*Executor)

func withColoniesServerHost(host string) ExecutorOption {
	return func(e *Executor) {
		e.coloniesServerHost = host
	}
}

func withColoniesServerPort(port int) ExecutorOption {
	return func(e *Executor) {
		e.coloniesServerPort = port
	}
}

func withColoniesInsecure(insecure bool) ExecutorOption {
	return func(e *Executor) {
		e.coloniesInsecure = insecure
	}
}

func withColonyName(name string) ExecutorOption {
	return func(e *Executor) {
		e.colonyName = name
	}
}

func withColonyPrvKey(prvkey string) ExecutorOption {
	return func(e *Executor) {
		e.colonyPrvKey = prvkey
	}
}

func withColonyID(id string) ExecutorOption {
	return func(e *Executor) {
		e.colonyID = id
	}
}

func withExecutorID(id string) ExecutorOption {
	return func(e *Executor) {
		e.executorID = id
	}
}

func withExecutorName(name string) ExecutorOption {
	return func(e *Executor) {
		e.executorName = name
	}
}

func withExecutorType(typ string) ExecutorOption {
	return func(e *Executor) {
		e.executorType = typ
	}
}

func withExecutorPrvKey(key string) ExecutorOption {
	return func(e *Executor) {
		e.executorPrvKey = key
	}
}

func (e *Executor) createExecutorwithKey(colonyName string) (*core.Executor, string, string, error) {
	crypto := crypto.CreateCrypto()
	executorPrvKey, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, "", "", err
	}

	executorID, err := crypto.GenerateID(executorPrvKey)
	if err != nil {
		return nil, "", "", err
	}

	executor := core.CreateExecutor(executorID, "cpm-anchor", e.executorName, colonyName, time.Now(), time.Now())

	return executor, executorID, executorPrvKey, nil
}

func CreateExecutor(opts ...ExecutorOption) (*Executor, error) {
	e := &Executor{}
	for _, opt := range opts {
		opt(e)
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.ctx = ctx
	e.cancel = cancel

	sigc := make(chan os.Signal)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSEGV)
	go func() {
		<-sigc
		e.Shutdown()
		os.Exit(1)
	}()

	e.client = client.CreateColoniesClient(e.coloniesServerHost, e.coloniesServerPort, e.coloniesInsecure, false)

	if e.colonyPrvKey != "" {
		spec, executorID, executorPrvKey, err := e.createExecutorwithKey(e.colonyName)
		if err != nil {
			return nil, err
		}
		e.executorID = executorID
		e.executorPrvKey = executorPrvKey

		_, err = e.client.AddExecutor(spec, e.colonyPrvKey)
		if err != nil {
			return nil, err
		}
		err = e.client.ApproveExecutor(e.colonyName, e.executorName, e.colonyPrvKey)
		if err != nil {
			return nil, err
		}

		function := &core.Function{ExecutorName: e.executorName, ColonyName: e.colonyName, FuncName: "sleep"}

		_, err = e.client.AddFunction(function, e.executorPrvKey)
		log.WithFields(log.Fields{"ExecutorID": e.executorID}).Info("Self-registered")
	}
	return e, nil
}

func (e *Executor) Shutdown() error {
	log.Info("Shutting down")
	if e.colonyPrvKey != "" {
		err := e.client.RemoveExecutor(e.colonyName, e.executorName, e.colonyPrvKey)
		if err != nil {
			log.WithFields(log.Fields{"ExecutorID": e.executorID}).Warning("Failed to deregister")
		}

		log.WithFields(log.Fields{"ExecutorID": e.executorID}).Info("Deregistered")
	}
	e.cancel()
	return nil
}

func (e *Executor) ServeForEver() error {
	for {
		process, err := e.client.AssignWithContext(e.colonyName, 100, e.ctx, "", "", e.executorPrvKey)
		if err != nil {
			var coloniesError *core.ColoniesError
			if errors.As(err, &coloniesError) {
				if coloniesError.Status == 404 { // No processes can be selected for executor
					log.Info(err)
					continue
				}
			}

			log.Error(err)
			log.Error("Retrying in 5 seconds ...")
			time.Sleep(5 * time.Second)

			continue
		}

		log.WithFields(log.Fields{"ProcessID": process.ID, "ExecutorID": e.executorID}).Info("Assigned process to executor")

		funcName := process.FunctionSpec.FuncName
		if funcName == "sleep" {
			if len(process.FunctionSpec.Args) != 1 {
				log.Info(err)
				err = e.client.Fail(process.ID, []string{"Invalid argument"}, e.executorPrvKey)
			}
			timeToSleepIf := process.FunctionSpec.Args[0]
			timeToSleepStr, ok := timeToSleepIf.(string)
			if !ok {
				log.Info(err)
				err = e.client.Fail(process.ID, []string{"Invalid argument, not string"}, e.executorPrvKey)
			}
			timeToSleep, err := strconv.Atoi(timeToSleepStr)
			if err != nil {
				log.Info(err)
				err = e.client.Fail(process.ID, []string{"Invalid argument, could not convert to int"}, e.executorPrvKey)
			}

			log.WithFields(log.Fields{"TimeToSleep": timeToSleep}).Info("Executing sleep function")

			time.Sleep(time.Duration(timeToSleep) * time.Millisecond)
			err = e.client.Close(process.ID, e.executorPrvKey)

			log.Info("Closing process")
		} else {
			log.WithFields(log.Fields{"ProcessID": process.ID, "ExecutorID": e.executorID, "FuncName": funcName}).Info("Unsupported function")
			err = e.client.Fail(process.ID, []string{"Unsupported function: " + funcName}, e.executorPrvKey)
			log.Info(err)
		}
	}
}

func main() {
	var (
		host           string
		port           int
		insecure       bool
		executorPrvKey string
		executorName   string
	)

	flag.StringVar(&host, "host", "localhost", "Colonies server host")
	flag.IntVar(&port, "port", 50080, "Colonies server port")
	flag.BoolVar(&insecure, "insecure", true, "Disable TLS")
	flag.StringVar(&executorPrvKey, "key", "", "Executor private key")
	flag.StringVar(&executorName, "name", "docker-spawn-executor", "Executor name")

	flag.Parse()

	e, err := CreateExecutor(
		withColoniesServerHost(host),
		withColoniesServerPort(port),
		withColoniesInsecure(insecure),
		withColonyPrvKey(executorPrvKey),
		withExecutorName(executorName),
		withColonyName("dev"),
		withExecutorID("1234"),
		withExecutorType("cpm-anchor"),
	)

	if err != nil {
		panic(err)
	}

	if err := e.ServeForEver(); err != nil {
		panic(err)
	}
}
