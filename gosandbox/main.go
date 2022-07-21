package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

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

	image, err := client.Pull(ctx, "docker.io/nikolabo/demowebapp:latest", containerd.WithPullUnpack)
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

	fmt.Println("Checkpointing")
	checkpoint, err := task.Checkpoint(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Checkpoint created")

	fmt.Print("Enter to restore:")
	scanner.Scan()

	fmt.Println("Restoring")
	demo, err := client.NewContainer(ctx, "demo", containerd.WithNewSnapshot("demo-rootfs", checkpoint))
	if err != nil {
		return err
	}
	defer demo.Delete(ctx, containerd.WithSnapshotCleanup)

	restoredtask, err := demo.NewTask(ctx, cio.NullIO, containerd.WithTaskCheckpoint(checkpoint))
	if err != nil {
		return err
	}
	defer restoredtask.Delete(ctx)

	if err := restoredtask.Start(ctx); err != nil {
		return err
	}
	fmt.Println("Restored")

	time.Sleep(2 * time.Second)

	if err := restoredtask.Kill(ctx, syscall.SIGKILL); err != nil {
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
