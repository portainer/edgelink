.PHONY: clean generate

clean:
	rm -rf keyx/api/v1/*.pb.go
	rm -rf examples/grpc/*.pb.go

generate: clean
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative keyx/api/v1/keyx.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative examples/grpc/grpc.proto
