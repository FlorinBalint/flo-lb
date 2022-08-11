package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	pb "github.com/FlorinBalint/flo_lb/proto"
	"google.golang.org/protobuf/encoding/prototext"
)

func Parse(cfg []byte, format pb.ConfigFormat) (*pb.Config, error) {
	res := &pb.Config{}
	var err error
	switch format {
	case pb.ConfigFormat_TEXT_PROTO:
		err = prototext.Unmarshal(cfg, res)
	case pb.ConfigFormat_JSON:
		err = fmt.Errorf("JSON is not supported yet")
	case pb.ConfigFormat_YAML:
		err = fmt.Errorf("YAML is not supported yet")
	}
	return res, err
}

func fileFormat(path string) (pb.ConfigFormat, error) {
	extension := path[strings.LastIndex(path, "."):]
	switch extension {
	case ".textpb", ".textproto", ".pb":
		return pb.ConfigFormat_TEXT_PROTO, nil
	case ".yaml":
		return pb.ConfigFormat_YAML, nil
	case ".json":
		return pb.ConfigFormat_JSON, nil
	default:
		return 0, fmt.Errorf("unknown extension format %v for %v, please add extension", extension, path)
	}
}

// Parse a load balancer config file
func ParseFile(path string) (*pb.Config, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	format, err := fileFormat(path)
	if err != nil {
		return nil, err
	}
	return Parse(content, format)
}
