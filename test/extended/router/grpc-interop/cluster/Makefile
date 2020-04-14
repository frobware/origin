BINARIES := bin/server bin/client

all: $(BINARIES)

clean:
	$(RM) $(BINARIES)

$(BINARIES): Makefile

bin/client: client/client.go
	go build -o $@ $<

bin/server: server/server.go
	go build -o $@ $<

.PHONY: all
