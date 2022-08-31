package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/remotes/docker/config"
)

var ctx context.Context

func main() {
	argLength := len(os.Args[1:])
	if argLength != 1 {
		log.Fatal("wrmcheckpt: missing container id argument")
	}

	containerId := os.Args[1]

	fmt.Println("Attempting to open containerd client connection...")
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	fmt.Println("Successfully opened containerd client connection!")

	ctx = namespaces.WithNamespace(context.Background(), "k8s.io")

	container, err := client.LoadContainer(ctx, containerId)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Checkpointing container %s...\n", containerId)
	imageStore := client.ImageService()
	checkpoint, err := container.Checkpoint(ctx, "wrmcheckpt", []containerd.CheckpointOpts{
		containerd.WithCheckpointRuntime,
		containerd.WithCheckpointRW,
		containerd.WithCheckpointTask,
	}...)
	if err != nil {
		log.Fatal(err)
	}
	defer imageStore.Delete(ctx, checkpoint.Name())
	fmt.Println("Checkpoint created")

	resolver := GetResolver()
	ropts := []containerd.RemoteOpt{containerd.WithResolver(resolver)}

	err = client.Push(ctx, "docker.io/nikolabo/io-checkpoint:latest", checkpoint.Target(), ropts...)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Checkpoint uploaded to remote")
}

// Sets up resolver with credentials to push to dockerhub
func GetResolver() remotes.Resolver {
	options := docker.ResolverOptions{}
	user := os.Getenv("user")
	access := os.Getenv("access")
	hostOptions := config.HostOptions{}
	hostOptions.Credentials = func(host string) (string, string, error) {
		return user, access, nil
	}

	options.Hosts = config.ConfigureHosts(ctx, hostOptions)
	return docker.NewResolver(options)
}
