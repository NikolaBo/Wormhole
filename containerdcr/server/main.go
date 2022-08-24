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
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var clientset *kubernetes.Clientset
var addr string
var host string

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
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

func configure(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("/configure endpoint accessed\n")

	addresses, ok := r.URL.Query()["addr"]
	if !ok || len(addresses[0]) < 1 {
		fmt.Fprintf(w, "Url Param 'addr' is missing\n")
		return
	}
	hosts, ok := r.URL.Query()["host"]
	if !ok || len(hosts[0]) < 1 {
		fmt.Fprintf(w, "Url Param 'host' is missing\n")
		return
	}
	addr = addresses[0]
	host = hosts[0]

	fmt.Fprintf(w, "Destination configured\n")
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

	// Setup pod definition
	pod := getPodObject()

	// Deploy pod
	pod, err = clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(),
		pod,
		metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Destination pod created successfully")
	fmt.Println(pod)

	fmt.Fprintf(w, "Checkpoint complete\n")
}

func getPodObject() *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "alprestr",
			Namespace: "default",
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "alpineio",
					Image:           "nikolabo/alpineio",
					ImagePullPolicy: core.PullIfNotPresent,
				},
			},
			NodeName: host,
		},
	}
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
