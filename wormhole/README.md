## Wormhole Pod
The Wormhole pod consists of a server application, checkpoint utility, and restore utility.

The wrm pod definition specifies resources required by a wormhole pod, such as a host filesystem volume, and takes environment variables as input to determine the pod host and container registry credentials.

The checkpoint and restore shell scripts transfer the utilities to the host filesystem and execute them in that context.