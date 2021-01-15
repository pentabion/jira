all: linux darwin windows freebsd

clean:
	rm -rf dist/darwin
	rm -rf dist/linux
	rm -rf dist/freebsd
	rm -rf dist/windows

get:
	go get github.com/andygrunwald/go-jira
	go get github.com/davecgh/go-spew/spew
	go get github.com/fatih/color

freebsd: get
	mkdir -p dist/freebsd
	GOOS=freebsd go build -o dist/freebsd/jira

linux: get
	mkdir -p dist/linux
	GOOS=linux go build -o dist/linux/jira

darwin: get
	mkdir -p dist/darwin
	GOOS=darwin go build -o dist/darwin/jira

windows: get
	mkdir -p dist/windows
	GOOS=windows go build -o dist/windows/jira
