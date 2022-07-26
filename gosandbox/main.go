package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
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

	ctx := namespaces.WithNamespace(context.Background(), "example")

	image, err := client.Pull(ctx, "docker.io/nikolabo/demowebps:latest", containerd.WithPullUnpack)
	if err != nil {
		return err
	}
	log.Printf("Successfully pulled %s image\n", image.Name())

	container, err := client.NewContainer(
		ctx,
		"demo-web-app",
		containerd.WithNewSnapshot("demo-web-app-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		return err
	}
	defer container.Delete(ctx, containerd.WithSnapshotCleanup)
	log.Printf("Successfully created container with ID %s and snapshot with ID demo-web-app-snapshot", container.ID())

	task, err := container.NewTask(ctx, cio.NullIO)
	if err != nil {
		return err
	}
	defer task.Delete(ctx)

	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		return err
	}

	if err := task.Start(ctx); err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter to checkpoint:")
	scanner.Scan()
	working, err := os.Getwd()
	if err != nil {
		return err
	}
	imagePath := filepath.Join(working, "cr")

	fmt.Println("Checkpointing")
	_, err = task.Checkpoint(ctx, containerd.WithCheckpointImagePath(imagePath))
	if err != nil {
		return err
	}

	if err = task.Kill(ctx, syscall.SIGTERM); err != nil {
		return err
	}

	<-exitStatusC

	if _, err = task.Delete(ctx); err != nil {
		return err
	}

	if err := container.Delete(ctx, containerd.WithSnapshotCleanup); err != nil {
		return err
	}

	fmt.Println("Checkpoint created")

	fmt.Print("Enter to restore:")
	scanner.Scan()

	fmt.Println("Restoring")

	demo, err := client.NewContainer(ctx, "demo", containerd.WithNewSnapshot("demo", image), containerd.WithNewSpec((oci.WithImageConfig(image))))
	if err != nil {
		return err
	}
	defer demo.Delete(ctx, containerd.WithSnapshotCleanup)

	restoredtask, err := demo.NewTask(ctx, cio.NullIO, containerd.WithRestoreImagePath(imagePath))
	if err != nil {
		return err
	}
	defer restoredtask.Delete(ctx)

	exitStatusC, err = restoredtask.Wait(ctx)
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
	fmt.Printf("demo-web-app exited with status: %d\n", code)

	return nil
}
