sudo ctr -n example t kill -s SIGKILL demo-web-app
sudo ctr -n example c rm demo-web-app
sudo ctr -n example i rm examplecheckpoint
sudo ctr c rm demo
sudo ctr snapshot rm demo
sudo ctr -n example content rm sha256:a9293ab5ff3cbdb36c0c37b7d9307125a46118dd61bc75438e060a911fd741db