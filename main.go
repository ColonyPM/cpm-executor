package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/security/crypto"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	dc "github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

var (
	host         string
	port         int
	insecure     bool
	colonyPrvKey string
	executorName string
)

func createContainer(_ *client.ColoniesClient, imgName string) (string, string, error) {
	ctx := context.Background()

	cli, err := dc.NewClientWithOpts(dc.FromEnv, dc.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	fmt.Printf("Pulling %s...\n", imgName)
	reader, err := cli.ImagePull(ctx, imgName, image.PullOptions{})
	if err != nil {
		return "", "", err
	}
	io.Copy(os.Stdout, reader)
	reader.Close()

	containerName := uuid.New().String()

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: imgName,
			Cmd: []string{
				host,
				strconv.Itoa(port),
				strconv.FormatBool(insecure),
				colonyPrvKey,
				containerName,
			},
		},
		&container.HostConfig{},
		nil,
		nil,
		containerName,
	)
	if err != nil {
		log.Error(err)
		return "", "", err
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", "", err
	}

	return containerName, resp.ID, nil
}

func removeContainer(_ *client.ColoniesClient, containerID string) error {
	ctx := context.Background()

	cli, err := dc.NewClientWithOpts(dc.FromEnv, dc.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	err = cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		RemoveVolumes: true,
	})

	if err != nil {
		return err
	}

	return nil
}

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

func CreateExecutor(
	host string,
	port int,
	insecure bool,
	colonyName string,
	colonyPrvKey string,
	executorName string,
	executorType string,
) (*Executor, error) {
	e := &Executor{
		coloniesServerHost: host,
		coloniesServerPort: port,
		coloniesInsecure:   insecure,
		colonyName:         colonyName,
		colonyPrvKey:       colonyPrvKey,
		executorName:       executorName,
		executorType:       executorType,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	e.ctx = ctx
	e.cancel = cancel

	go func() {
		<-ctx.Done()
		e.Shutdown()
		os.Exit(0)
	}()

	e.client = client.CreateColoniesClient(e.coloniesServerHost, e.coloniesServerPort, e.coloniesInsecure, false)

	crypto := crypto.CreateCrypto()

	executorPrvKey, err := crypto.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	executorID, err := crypto.GenerateID(executorPrvKey)
	if err != nil {
		return nil, err
	}

	e.executorPrvKey = executorPrvKey
	e.executorID = executorID

	spec := core.CreateExecutor(e.executorID, e.executorType, e.executorName, e.colonyName, time.Now(), time.Now())
	if _, err = e.client.AddExecutor(spec, e.colonyPrvKey); err != nil {
		return nil, err
	}

	if err = e.client.ApproveExecutor(e.colonyName, e.executorName, e.colonyPrvKey); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Executor) Shutdown() error {
	log.Info("Shutting down")

	err := e.client.RemoveExecutor(e.colonyName, e.executorName, e.colonyPrvKey)
	if err != nil {
		log.WithFields(log.Fields{"ExecutorID": e.executorID}).Warning("Failed to deregister")
	}

	log.WithFields(log.Fields{"ExecutorID": e.executorID}).Info("Deregistered")
	e.cancel()

	return nil
}

func (e *Executor) ServeForEver() error {
	for {
		process, err := e.client.AssignWithContext(e.colonyName, 100, e.ctx, "", "", e.executorPrvKey)
		if err != nil {
			var coloniesError *core.ColoniesError
			if errors.As(err, &coloniesError) {
				if coloniesError.Status == 404 {
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
		if funcName == "createExecutor" {
			if len(process.FunctionSpec.Args) != 1 {
				if err = e.client.Fail(process.ID, []string{"missing imgName argument"}, e.executorPrvKey); err != nil {
					log.Info(err)
				}

				continue
			}

			imgName, ok := process.FunctionSpec.Args[0].(string)
			if !ok {
				if err = e.client.Fail(process.ID, []string{"could not convert imgName argument to a string"}, e.executorPrvKey); err != nil {
					log.Info(err)
				}

				continue
			}

			containerName, containerID, err := createContainer(e.client, imgName)
			if err != nil {
				if err = e.client.Fail(process.ID, []string{err.Error()}, e.executorPrvKey); err != nil {
					log.Info(err)
					os.Exit(1)
				}
			}

			err = e.client.CloseWithOutput(process.ID, []any{containerName, containerID}, e.executorPrvKey)
			fmt.Println(err)

			log.Info("Closing process")
		} else if funcName == "removeExecutor" {
			if len(process.FunctionSpec.Args) != 1 {
				if err = e.client.Fail(process.ID, []string{"missing containerID argument"}, e.executorPrvKey); err != nil {
					log.Info(err)
				}

				continue
			}

			containerID, ok := process.FunctionSpec.Args[0].(string)
			if !ok {
				if err = e.client.Fail(process.ID, []string{"could not convert containerID argument to a string"}, e.executorPrvKey); err != nil {
					log.Info(err)
				}

				continue
			}

			err := removeContainer(e.client, containerID)
			if err != nil {
				if err = e.client.Fail(process.ID, []string{err.Error()}, e.executorPrvKey); err != nil {
					log.Info(err)
					os.Exit(1)
				}
			}

			err = e.client.Close(process.ID, e.executorPrvKey)
			fmt.Println(err)

			log.Info("Closing process")
		} else {
			log.WithFields(log.Fields{"ProcessID": process.ID, "ExecutorID": e.executorID, "FuncName": funcName}).Info("Unsupported function")
			err = e.client.Fail(process.ID, []string{fmt.Sprintf("unsupported function '%s'", funcName)}, e.executorPrvKey)
			log.Info(err)
		}
	}
}

func main() {
	flag.StringVar(&host, "host", "localhost", "Colonies server host")
	flag.IntVar(&port, "port", 50080, "Colonies server port")
	flag.BoolVar(&insecure, "insecure", true, "Disable TLS")
	flag.StringVar(&colonyPrvKey, "key", "", "Colony private key")
	flag.StringVar(&executorName, "name", "", "Executor name")
	flag.Parse()

	e, err := CreateExecutor(
		host,
		port,
		insecure,
		"dev",
		colonyPrvKey,
		executorName,
		"cpm-anchor",
	)

	if err != nil {
		log.Fatalf("Failed to initialize executor: %v", err)
	}

	fmt.Println("Anchor started...")

	if err := e.ServeForEver(); err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
}
