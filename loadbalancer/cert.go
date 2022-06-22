package loadbalancer

import (
	"context"
	"crypto/tls"
	"log"
)

// load TLS configuration
func (s *Server) SetupTLS(ctx context.Context) error {
	certCfg := s.cfg.GetCert()

	if certCfg.GetAcme() != nil {
		log.Fatalf("TODO(#1): Implement automatic certificate management")
	}

	certPath := certCfg.GetLocal().GetCertPath()
	keyPath := certCfg.GetLocal().GetPrivateKeyPath()
	cer, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return err
	}

	s.server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cer}}
	return nil
}
