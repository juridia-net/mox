#!/bin/bash
set -e

REMOTE=root@100.113.21.6
REMOTE_PATH=/home/mox/
SERVICE=mox

# Stop mox service on remote
ssh $REMOTE "systemctl stop $SERVICE"

# Copy mox binary to remote
scp ./mox $REMOTE:$REMOTE_PATH

# Start mox service on remote
ssh $REMOTE "systemctl start $SERVICE"

echo "Deployment complete." 