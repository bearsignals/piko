BINARY_NAME=piko
INSTALL_PATH=/usr/local/bin

.PHONY: build install clean

build:
	go build -o $(BINARY_NAME) ./cmd/piko

install: build
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	rm -f $(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
