go: main.go 
	# ------------------------------------------------------
	# build the go executable
	mkdir -p ${PWD}/tmpbuild
	GOPATH=${PWD}/tmpbuild go get github.com/gorilla/websocket
	GOPATH=${PWD}/tmpbuild go get github.com/timtadh/getopt
	GOPATH=${PWD}/tmpbuild go build -o eyes-go-server .

lint:
	# ------------------------------------------------------
	# go linter
	mkdir -p ${PWD}/tmpbuild
	GOPATH=${PWD}/tmpbuild go get github.com/gorilla/websocket
	GOPATH=${PWD}/tmpbuild go get github.com/timtadh/getopt
	# running linter for GO
	GOPATH=${PWD}/tmpbuild ../bin/golangci-lint run ./*.go

build: go lint 

run: go foreground

foreground:
	./eyes-go-server
    
