package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/remotes/docker/config"
)

var client *containerd.Client
var ctx context.Context

func main() {
	fmt.Printf("Attempting to open containerd client connection...\n")
	var err error
	client, err = containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	fmt.Printf("Successfully opened containerd client connection!\n")

	ctx = namespaces.WithNamespace(context.Background(), "k8s.io")

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello!\n")
		fmt.Printf("/hello endpoint accessed\n")
	})

	http.HandleFunc("/checkpoint", createCheckpoint)

	fmt.Printf("Starting server at port 8080\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func createCheckpoint(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("/checkpoint endpoint accessed\n")

	ids, ok := r.URL.Query()["id"]
	if !ok || len(ids[0]) < 1 {
		fmt.Fprintf(w, "Url Param 'id' is missing\n")
		return
	}
	containerId := ids[0]

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
	fmt.Printf("Checkpoint created\n")

	resolver := GetResolver()
	ropts := []containerd.RemoteOpt{containerd.WithResolver(resolver)}

	err = client.Push(ctx, "docker.io/nikolabo/io-checkpoint:latest", checkpoint.Target(), ropts...)
	if err != nil {
		log.Fatal(err)
	}
}

func GetResolver() remotes.Resolver {
	options := docker.ResolverOptions{}
	user := ""
	access := ""
	hostOptions := config.HostOptions{}
	hostOptions.Credentials = func(host string) (string, string, error) {
		return user, access, nil
	}

	options.Hosts = config.ConfigureHosts(ctx, hostOptions)
	return docker.NewResolver(options)
}

func restoreFromCheckpoint() error {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	scanner := bufio.NewScanner(os.Stdin)

	ctx := namespaces.WithNamespace(context.Background(), "default")

	imageStore := client.ImageService()
	image, err := imageStore.Get(ctx, "docker.io/nikolabo/io-checkpoint:latest")
	if err != nil {
		return err
	}
	checkpoint := containerd.NewImage(client, image)

	containerName := "demo"
	argLength := len(os.Args[1:])
	if argLength != 0 {
		containerName = os.Args[1]
	}

	demo, err := client.Restore(ctx, containerName, checkpoint, []containerd.RestoreOpts{
		containerd.WithRestoreImage,
		containerd.WithRestoreSpec,
		containerd.WithRestoreRuntime,
		containerd.WithRestoreRW,
	}...)
	if err != nil {
		return err
	}
	defer demo.Delete(ctx, containerd.WithSnapshotCleanup)

	restoredtask, err := demo.NewTask(ctx, cio.NullIO, containerd.WithTaskCheckpoint(checkpoint))
	if err != nil {
		return err
	}
	defer restoredtask.Delete(ctx)

	exitStatusC, err := restoredtask.Wait(ctx)
	if err != nil {
		return err
	}

	if err := restoredtask.Start(ctx); err != nil {
		return err
	}
	fmt.Println("Restored")

	fmt.Print("Enter to kill:")
	scanner.Scan()

	if err := restoredtask.Kill(ctx, syscall.SIGTERM); err != nil {
		return err
	}

	status := <-exitStatusC
	code, _, err := status.Result()
	if err != nil {
		return err
	}
	fmt.Printf("%s exited with status: %d\n", demo.ID(), code)

	return nil
}
