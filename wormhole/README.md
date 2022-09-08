## Wormhole Pod
The Wormhole pod consists of a server application, checkpoint utility, and restore utility.

The wrm pod definition specifies resources required by a wormhole pod, such as a host filesystem volume, and takes environment variables as input to determine the pod host and container registry credentials.

The checkpoint and restore shell scripts transfer the utilities to the host filesystem and execute them in that context.

## Building and Running
In `./server`: `go build main.go`.

In `./checkpt`: `go build checkpoint.go`.

In `./restore`: `go build restore.go`.

`docker build`

Use `docker push` to add image to  a registry.

Deploy to cluster with: `kubectl apply -f wrm.yaml`. Can use `envsubst` to supply node hostname etc.