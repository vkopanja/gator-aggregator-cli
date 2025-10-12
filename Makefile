.PHONY: build clean

build:
	go build -o bin/gator .
clean:
	rm -rf bin/