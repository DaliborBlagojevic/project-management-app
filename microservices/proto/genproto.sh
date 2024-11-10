protoc \
    --go_out=./user --go_opt=paths=source_relative \
    --go-grpc_out=./user --go-grpc_opt=paths=source_relative \
    user_service.proto