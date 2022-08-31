## Wormhole Server
The server handles incoming requests, and is a client of the Kubernetes API.

The configure endpoint notifies the server of the destination for future migrations.

The migrate endpoint creates and uploads a checkpoint, deletes the source pod, creates the destination pod and sends a restore request to the destination.

The restore endpoint fetches the checkpoint image and restores a container into the destination pod.