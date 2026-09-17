module taskmanager/service/task

go 1.25.0

replace taskmanager/common => ../../common

require (
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0
	go.uber.org/fx v1.24.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260904194346-d0f1323225a4
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.12
	taskmanager/common v0.0.0-00010101000000-000000000000
)

require (
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260911204522-f61a6ca850bd // indirect
)
