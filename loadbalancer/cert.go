package loadbalancer

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	pb "github.com/FlorinBalint/flo_lb/proto"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

func (s *Server) setTLSFromLocalFiles(localCfg *pb.LocalCert) error {
	cert := localCfg.GetCertPath()
	key := localCfg.GetPrivateKeyPath()

	if len(cert) == 0 || len(key) == 0 {
		return fmt.Errorf("Local setup must specify the certificate and key path")
	}
	cer, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}

	s.server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cer}}
	return nil
}

func (s *Server) setAutomaticTLS(acmeCfg *pb.AcmeCert) error {
	domain := acmeCfg.GetDomain()
	serverDirURL := acmeCfg.GetServerDir()
	if len(domain) == 0 || len(serverDirURL) == 0 {
		return fmt.Errorf("Automatic certificate management requires the domain and the server directory to be set.")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("error generating key: %v", err)
	}

	acmeClient := &acme.Client{
		Key:          key,
		DirectoryURL: serverDirURL,
	}
	manager := &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		HostPolicy:  autocert.HostWhitelist(domain),
		Client:      acmeClient,
		RenewBefore: 24 * time.Hour,
	}
	if acmeCfg.GetCacheDirectory() != "" {
		manager.Cache = autocert.DirCache(acmeCfg.GetCacheDirectory())
	}

	tlsConfig := manager.TLSConfig()
	tlsConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert, err := manager.GetCertificate(hello)
		if err != nil {
			log.Printf("Error getting certificates %v", err)
		}
		return cert, err
	}
	s.server.TLSConfig = tlsConfig
	return nil
}

// load TLS configuration
func (s *Server) SetupTLS(ctx context.Context) error {
	certCfg := s.cfg.GetCert()

	if certCfg.GetAcme() != nil {
		return s.setAutomaticTLS(certCfg.GetAcme())
	}
	return s.setTLSFromLocalFiles(certCfg.GetLocal())
}
