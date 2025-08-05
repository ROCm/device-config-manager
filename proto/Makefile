.PHONY: bld_proto
bld_proto:
	@echo "building proto"
	@protoc --proto_path=. --go-grpc_out=../ --go_out=../ $(shell ls *.proto)

.PHONY: all
all:
	${MAKE} bld_proto