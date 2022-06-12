module github.com/FlorinBalint/flo_lb

go 1.18

require (
	github.com/FlorinBalint/flo_lb/proto v0.1.0
	google.golang.org/protobuf v1.28.0
)

replace github.com/FlorinBalint/flo_lb/proto => ./build/src/flo_lb/proto
