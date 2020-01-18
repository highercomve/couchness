# Usage:
# make        # compile all binary
# make clean  # remove ALL binaries and objects

.PHONY = all clean

all: build optimize

pre-build:
	pkger -o package

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/linux_amd64/couchness -v .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -ldflags="-s -w" -o build/linux_arm/couchness -v .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o build/darwin_amd64/couchness -v .
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -o build/windows_386/couchness.exe -v .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/windows_amd64/couchness.exe -v .

optimize:
	upx build/linux_amd64/couchness
	upx build/linux_arm/couchness

clean:
	@echo "Cleaning up..."
	rm -rf build
	rm -rf package