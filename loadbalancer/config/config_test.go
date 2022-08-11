package config

import (
	"path"
	"runtime"
	"testing"

	pb "github.com/FlorinBalint/flo_lb/proto"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
)

var wantProto = &pb.Config{
	Name: proto.String("flo_lb"),
	Port: proto.Int32(443),
	Backend: &pb.BackendConfig{
		Type: &pb.BackendConfig_Dynamic{
			Dynamic: &pb.DynamicBackends{
				RegisterPath:   proto.String("/register"),
				DeregisterPath: proto.String("/deregister"),
			},
		},
	},
	Protocol: pb.Protocol_HTTPS.Enum(),
	Cert: &pb.CertConfig{
		CertSource: &pb.CertConfig_Acme{
			Acme: &pb.AcmeCert{
				Domain:    "florinbalint.com",
				ServerDir: "https://acme-v02.api.letsencrypt.org/directory",
			},
		},
	},
	HealthCheck: &pb.HealthCheck{
		Probe: &pb.HealthProbe{
			Type: &pb.HealthProbe_HttpGet{
				HttpGet: &pb.HttpGet{
					Path: proto.String("/healthz"),
				},
			},
		},
		InitialDelay: &durationpb.Duration{
			Seconds: 10,
		},
		Period: &durationpb.Duration{
			Seconds: 5,
		},
		DisconnectThreshold: proto.Int32(5),
	},
}

func testFile(testFile string) string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Join(path.Dir(filename), "/testdata/", testFile)
}

func TestParseString(t *testing.T) {
	input := `
	name: "flo_lb"
port: 443
backend {
  dynamic {
    register_path: "/register"
    deregister_path: "/deregister"
  }
}
protocol: HTTPS
cert {
  acme {
    domain: "florinbalint.com"
    server_dir: "https://acme-v02.api.letsencrypt.org/directory"
  }
}
health_check {
  probe {
    http_get {
      path: "/healthz"
    }
  }
  initial_delay {
    seconds: 10
  }
  period {
    seconds: 5
  }
  disconnect_threshold: 5
}`
	got, err := Parse([]byte(input), pb.ConfigFormat_TEXT_PROTO)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if diff := cmp.Diff(wantProto, got, protocmp.Transform()); diff != "" {
		t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
	}
}

func TestParseFilePB(t *testing.T) {
	file := testFile("test_config.textproto")
	got, err := ParseFile(file)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if diff := cmp.Diff(wantProto, got, protocmp.Transform()); diff != "" {
		t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
	}
}

func TestParseFileYAML(t *testing.T) {
	// TODO
}

func TestParseFileJSON(t *testing.T) {
	file := testFile("test_config.json")
	got, err := ParseFile(file)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if diff := cmp.Diff(wantProto, got, protocmp.Transform()); diff != "" {
		t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
	}
}
