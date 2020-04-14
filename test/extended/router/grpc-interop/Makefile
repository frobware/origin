TARGET := ../../testdata/router/router-grpc-interop.yaml

all: $(TARGET)

$(TARGET): gen.go
	go run gen.go > $@

gen.go: Makefile build-check

build-check:
	@$(MAKE) -C cluster

update-bindata: ../../testdata/router/router-grpc-interop.yaml
	(cd ../../../..; ./hack/update-generated-bindata.sh)

clean:
	@$(RM) -f $(TARGET)
	@$(MAKE) -C cluster $@

.PHONY: \
	all \
	clean \
	build-check
