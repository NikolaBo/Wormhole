package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	// get pods in all the namespaces by omitting namespace
	// Or specify namespace to get pods in particular namespace
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	// Examples for error handling:
	// - Use helper functions e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Pods("default").Get(context.TODO(), "example-xxxxx", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Println("Pod example-xxxxx not found in default namespace")
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found example-xxxxx pod in default namespace\n")
	}

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello!\n")
		fmt.Println("/hello endpoint accessed")
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

	cmd := exec.Command("./checkpoint.sh", containerId)
	stdout, err := cmd.Output()
	fmt.Println(string(stdout[:]))
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Fprintf(w, "Checkpoint complete\n")
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
