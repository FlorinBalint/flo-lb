package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	pb "github.com/FlorinBalint/flo_lb/proto"
	xml2json "github.com/basgys/goxml2json"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"sigs.k8s.io/yaml"
)

func Parse(cfg []byte, format pb.ConfigFormat) (*pb.Config, error) {
	res := &pb.Config{}
	var err error
	switch format {
	case pb.ConfigFormat_TEXT_PROTO:
		err = prototext.Unmarshal(cfg, res)
	case pb.ConfigFormat_JSON:
		err = protojson.Unmarshal(cfg, res)
	case pb.ConfigFormat_YAML:
		json, err := yaml.YAMLToJSON(cfg)
		if err != nil {
			return nil, err
		}
		err = protojson.Unmarshal(json, res)
	case pb.ConfigFormat_XML:
		xml := strings.NewReader(string(cfg))
		json, err := xml2json.Convert(xml)
		if err != nil {
			return nil, err
		}
		err = protojson.Unmarshal(json.Bytes(), res)
	}
	return res, err
}

func fileFormat(path string) (pb.ConfigFormat, error) {
	extension := path[strings.LastIndex(path, "."):]
	switch extension {
	case ".textpb", ".textproto", ".pb":
		return pb.ConfigFormat_TEXT_PROTO, nil
	case ".yaml", ".yml":
		return pb.ConfigFormat_YAML, nil
	case ".json":
		return pb.ConfigFormat_JSON, nil
	case ".xml":
		return pb.ConfigFormat_XML, nil
	default:
		return 0, fmt.Errorf("unknown extension format %v for %v, please add extension", extension, path)
	}
}

// Parse a load balancer config file
func ParseFile(path string) (*pb.Config, error) {
	format, err := fileFormat(path)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(content, format)
}
