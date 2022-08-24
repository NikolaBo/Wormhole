package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/typeurl"
)

var ctx context.Context
var rules = []string{`/pods/(.*)/etc-hosts`,
	`/alpineio/(.*)"`,
	`/sandboxes/(.*)/`,
	`access-(.*)"`,
	`/besteffort/.*/(.*)"`,
	`/proc/(.*)/ns/`}

func main() {
	argLength := len(os.Args[1:])
	if argLength != 1 {
		log.Fatal("wrmrestore: missing container id argument")
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

	image, err := client.Pull(ctx, "docker.io/nikolabo/io-checkpoint:latest")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Successfully pulled %s image\n", image.Name())

	d := string(image.Metadata().Target.Digest)
	manifest, err := ioutil.ReadFile("/var/lib/containerd/io.containerd.content.v1.content/blobs/sha256/" + d[7:len(d)])
	if err != nil {
		log.Fatal(err)
	}
	re := regexp.MustCompile(`checkpoint\.config[^:]*:[^:]*:([^"]*)"`)
	specDigest := string(re.FindSubmatch(manifest)[1])

	spec, err := ioutil.ReadFile("/var/lib/containerd/io.containerd.content.v1.content/blobs/sha256/" + specDigest)

	container, err := client.LoadContainer(ctx, containerId)
	if err != nil {
		log.Fatal(err)
	}
	info, err := container.Info(ctx, containerd.WithoutRefreshedMetadata)
	v, err := typeurl.UnmarshalAny(info.Spec)
	x := struct {
		containers.Container
		Spec interface{} `json:"Spec,omitempty"`
	}{
		Container: info,
		Spec:      v,
	}
	in, err := json.Marshal(x)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't marshal %+v as a JSON string: %v\n", x, err)
	}

	// in, _ := ioutil.ReadFile("in")
	fmt.Println(string(spec))

	for _, exp := range rules {
		re = regexp.MustCompile(exp)
		fmt.Println(exp)
		new := re.FindSubmatch(in)[1]
		old := re.FindSubmatch(spec)[1]
		spec = []byte(strings.ReplaceAll(string(spec), string(old), string(new)))
	}

	fmt.Println(string(spec))
}
