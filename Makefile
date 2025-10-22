.PHONY: all server client clients test clean

all: server client local_terraform firestore_tester

server:
	@echo "Building server executable..."
	go build -o server ./cmd/server

client:
	@echo "Building client executable..."
	go build -o client ./cmd/client

local_terraform:
	@echo "Building local_terraform (load tester) executable..."
	go build -o local_terraform ./cmd/local_terraform

firestore_tester:
	@echo "Building firestore_tester executable..."
	go build -o firestore_tester ./cmd/firestore_tester

clients: client local_terraform firestore_tester

test: local_terraform
	@echo "Running local test..."
	./local_terraform

clean:
	@echo "Cleaning up binaries..."
	rm -f server client local_terraform firestore_tester

