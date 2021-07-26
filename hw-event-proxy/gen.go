package proxy

// This is a tool for generating message parser client located under `pb` directory

//go:generate mkdir -p pb
//go:generate protoc --go_out=plugins=grpc:pb --go_opt=paths=source_relative message_parser.proto
