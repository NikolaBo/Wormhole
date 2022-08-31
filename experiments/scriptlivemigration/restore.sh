cr=$(docker create --name kvstore -p 8000:8000 nikolabo/demowebapp)
cp -r /home/mariner_user/checkpoints/check /var/lib/docker/containers/$cr/checkpoints/check
docker start --checkpoint check kvstore
