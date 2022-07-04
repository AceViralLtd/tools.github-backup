.PHONY: build
build:
	CGO_ENABLED=0 go build -o ./ ./...

.PHONY: run
run:
	go run ./cmd/github-backup 

.PHONY: clean
clean:
	rm ./github-backup
	rm ./backup-respore

