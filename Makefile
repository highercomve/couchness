# Usage:
# make        # compile all binary
# make clean  # remove ALL binaries and objects

.PHONY = all clean

all: build

pre-build:
	pkger -o package

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/linux_amd64/couchness -v .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -o build/linux_arm/couchness -v .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/darwin_amd64/couchness -v .
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -o build/windows_386/couchness.exe -v .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/windows_amd64/couchness.exe -v .

clean:
	@echo "Cleaning up..."
	rm -rf build
	rm -rf package