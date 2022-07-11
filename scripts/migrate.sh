ipaddr=""
docker checkpoint create kvstore check
scp -r /var/lib/docker/containers/$1/checkpoints/check $ipaddr:/home/mariner_user/checkpoints
ssh $ipaddr 'echo | sudo -S /home/mariner_user/restore.sh' &> /dev/null
