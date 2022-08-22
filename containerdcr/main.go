package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
)

func main() {
	if err := webAppExample(); err != nil {
		log.Fatal(err)
	}
}

func webAppExample() error {
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
