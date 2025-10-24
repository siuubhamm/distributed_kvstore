#!/bin/bash

set -e
set -x

# --- Configuration ---
PROJECT_ID="fa25-bl-engr-e516-sk213"
ZONE="us-central1-a"
NETWORK_NAME="kvstore-network"

SERVER_VM_NAME="kvstore-server"
TAG="memcached-server"
CLIENT_VM_NAME="kvstore-client-1"

MACHINE_TYPE="e2-micro"
IMAGE_FAMILY="debian-11"
IMAGE_PROJECT="debian-cloud"

GIT_REPO_URL="https://github.com/siuubhamm/distributed_kvstore.git"
REPO_DIR_NAME="distributed_kvstore"
GO_VERSION="1.25.0"

SSH_KEY_FILE="/mnt/c/Users/shubh/.ssh/gcp_key.pub"

# --- Validation ---
if [ ! -f "$SSH_KEY_FILE" ]; then
    echo "ERROR: SSH key file not found at $SSH_KEY_FILE"
    exit 1
fi

echo "--- Starting VM Creation for 1 server and 1 client... ---"
gcloud config set project $PROJECT_ID

# --- Create Server VM ---
echo "Creating server VM: $SERVER_VM_NAME..."
gcloud compute instances create $SERVER_VM_NAME \
    --zone=$ZONE \
    --machine-type=$MACHINE_TYPE \
    --image-family=$IMAGE_FAMILY \
    --image-project=$IMAGE_PROJECT \
    --network=$NETWORK_NAME \
    --tags=$TAG \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --metadata-from-file=ssh-keys=$SSH_KEY_FILE \
    --quiet

# --- Create Client VM ---
echo "Creating client VM: $CLIENT_VM_NAME..."
gcloud compute instances create $CLIENT_VM_NAME \
    --zone=$ZONE \
    --machine-type=$MACHINE_TYPE \
    --image-family=$IMAGE_FAMILY \
    --image-project=$IMAGE_PROJECT \
    --network=$NETWORK_NAME \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --metadata-from-file=ssh-keys=$SSH_KEY_FILE \
    --quiet

echo "--- Both VMs Created. Waiting 60s for them to be ready... ---"
sleep 60

# --- 1. Deploy Server Software ---
echo "--- Connecting to $SERVER_VM_NAME to deploy server... ---"
gcloud compute ssh $SERVER_VM_NAME --zone=$ZONE --command="bash --noprofile --norc -c '
    set -e
    set -x
    echo \"[VM-SERVER] Updating packages...\"
    sudo apt update
    sudo apt install -y git build-essential wget

    echo \"[VM-SERVER] Installing Go...\"
    wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    export PATH=\$PATH:/usr/local/go/bin
    echo \"export PATH=\$PATH:/usr/local/go/bin\" >> ~/.bashrc
    go version

    echo \"[VM-SERVER] Cloning repo and building server...\"
    git clone $GIT_REPO_URL
    cd $REPO_DIR_NAME
    go mod tidy
    make server

    echo \"[VM-SERVER] Starting server in background...\"
    nohup ./server &> server.log &
    echo \"[VM-SERVER] Deployment complete. Server is running.\"
'"

# --- 2. Get Server's Internal IP ---
echo "--- Fetching Server's Internal IP... ---"
INTERNAL_IP=$(gcloud compute instances describe $SERVER_VM_NAME --zone=$ZONE --format='value(networkInterfaces[0].networkIP)')
echo "Server Internal IP found: $INTERNAL_IP"

# --- 3. Deploy Client Software ---
echo "--- Connecting to $CLIENT_VM_NAME to deploy client tools... ---"
gcloud compute ssh $CLIENT_VM_NAME --zone=$ZONE --command="bash --noprofile --norc -c '
    set -e
    set -x
    echo \"[$CLIENT_VM_NAME] Updating packages...\"
    sudo apt update
    sudo apt install -y git build-essential wget

    echo \"[$CLIENT_VM_NAME] Installing Go...\"
    wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    export PATH=\$PATH:/usr/local/go/bin
    echo \"export PATH=\$PATH:/usr/local/go/bin\" >> ~/.bashrc
    go version

    echo \"[$CLIENT_VM_NAME] Cloning repo and building clients...\"
    git clone $GIT_REPO_URL
    cd $REPO_DIR_NAME
    go mod tidy
    make clients
    echo \"[$CLIENT_VM_NAME] Client tools are built and ready.\"
'"

# --- 4. Run All Tests ---
echo "--- All software deployed. Running performance tests... ---"
echo "--- Starting tests from $CLIENT_VM_NAME... ---"
gcloud compute ssh $CLIENT_VM_NAME --zone=$ZONE --command="bash --noprofile --norc -c '
    set -e
    set -x
    export PATH=\$PATH:/usr/local/go/bin
    cd $REPO_DIR_NAME

    echo \"[$CLIENT_VM_NAME] RUNNING: ./local_terraform test against $INTERNAL_IP\"
    ./local_terraform -server=$INTERNAL_IP:8080

    echo \"[$CLIENT_VM_NAME] RUNNING: ./firestore_tester test\"
    ./firestore_tester
'"

echo "--- All tests complete. ---"

# --- 5. Download persistent.json ---
echo "--- Downloading persistent.json from server... ---"
gcloud compute scp ${SERVER_VM_NAME}:~/${REPO_DIR_NAME}/persistent.json . --zone=$ZONE
echo "--- persistent.json downloaded successfully. ---"