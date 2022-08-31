package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/containerd/typeurl"
)

var ctx context.Context

// Regex rules for pod-specific resources
var rules = []string{
	`/pods/(.*)/etc-hosts`, // pod uid
	`/workload/(.*)"`,      // termination log id
	`/sandboxes/(.*)/`,     // cri pod sandbox id
	`access-(.*)"`,         // kube api access id
	`/besteffort/.*/(.*)"`, // container id
	`/proc/(.*)/ns/`}       // pod namesace pid

func main() {
	containerName := "alprestr"
	argLength := len(os.Args[1:])
	if argLength == 2 {
		containerName = os.Args[2]
	} else if argLength != 1 {
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

	// Pull image
	image := imagePull(ctx, client)
	fmt.Printf("Successfully pulled %s image\n\n", image.Name())

	// Extract spec that needs to be modified
	d := string(image.Metadata().Target.Digest)
	manifest, err := ioutil.ReadFile("/var/lib/containerd/io.containerd.content.v1.content/blobs/sha256/" + d[7:len(d)])
	if err != nil {
		log.Fatal(err)
	}
	re := regexp.MustCompile(`checkpoint\.config[^:]*:[^:]*:([^"]*)"`)
	specDigest := string(re.FindSubmatch(manifest)[1])
	specPath := "/var/lib/containerd/io.containerd.content.v1.content/blobs/sha256/" + specDigest

	spec, err := ioutil.ReadFile(specPath)
	if err != nil {
		log.Fatal(err)
	}
	stat, err := os.Stat(specPath)
	if err != nil {
		log.Fatal(err)
	}
	fileSize := stat.Size()

	// Extract spec of destination pod container
	container, err := client.LoadContainer(ctx, containerId)
	if err != nil {
		log.Fatal(err)
	}
	info, err := container.Info(ctx, containerd.WithoutRefreshedMetadata)
	if err != nil {
		log.Fatal(err)
	}
	v, err := typeurl.UnmarshalAny(info.Spec)
	if err != nil {
		log.Fatal(err)
	}
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

	// Modify spec to match new pod external resources
	for _, exp := range rules {
		re = regexp.MustCompile(exp)
		new := re.FindSubmatch(in)[1]
		old := re.FindSubmatch(spec)[1]
		spec = []byte(strings.ReplaceAll(string(spec), string(old), string(new)))
	}

	// If modified spec is shorter, add whitespace bytes
	// Needed to satisfy containerd image parsing
	for len(spec) < int(fileSize) {
		spec = []byte(string(spec) + " ")
	}
	ioutil.WriteFile("/var/lib/containerd/io.containerd.content.v1.content/blobs/sha256/"+specDigest, spec, fs.ModeTemporary)

	// Restore container from checkpoint
	checkpoint := image
	restored, err := client.Restore(ctx, containerName, checkpoint, []containerd.RestoreOpts{
		containerd.WithRestoreImage,
		containerd.WithRestoreSpec,
		containerd.WithRestoreRuntime,
		containerd.WithRestoreRW,
	}...)
	if err != nil {
		log.Fatal(err)
	}

	restoredtask, err := restored.NewTask(ctx, cio.NullIO, containerd.WithTaskCheckpoint(checkpoint))
	if err != nil {
		log.Fatal(err)
	}
	defer restoredtask.Delete(ctx)

	if err := restoredtask.Start(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Restored")
}

// Fetch and unpack checkpoint image from container registry
// Based on implementation in containerd ctr cli: https://github.com/containerd/containerd/blob/1bb39b833e26a6dc7435fcec8ded44a04b3827f7/cmd/ctr/commands/images/pull.go#L70
func imagePull(ctx context.Context, client *containerd.Client) containerd.Image {
	image, err := client.Fetch(ctx, "docker.io/nikolabo/io-checkpoint:latest")
	if err != nil {
		log.Fatal(err)
	}

	p, err := images.Platforms(ctx, client.ContentStore(), image.Target)
	if err != nil {
		log.Fatal("unable to resolve image platforms: %w", err)
	}

	if len(p) == 0 {
		p = append(p, platforms.DefaultSpec())
	}

	for _, platform := range p {
		fmt.Printf("unpacking %s %s...\n", platforms.Format(platform), image.Target.Digest)
		i := containerd.NewImageWithPlatform(client, image, platforms.Only(platform))
		err = i.Unpack(ctx, "")
		if err != nil {
			fmt.Println(err)
		}
	}
	return containerd.NewImage(client, image)
}
