.PHONY: build clean

build:
	go build -o bin/fc-explorer .

clean:
	rm -rf bin/
