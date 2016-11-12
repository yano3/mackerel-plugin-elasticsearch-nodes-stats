deps:
	go get -d -t ./...

test: deps
	go test -v

build: deps
	gox -osarch="linux/amd64" -output="pkg/{{.OS}}_{{.Arch}}/{{.Dir}}"

lint:
	golint ./...
