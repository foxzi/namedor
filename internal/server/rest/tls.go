package rest

import (
	"crypto/tls"
	"log"
	"sync"
	"time"
)

// certReloader handles automatic reloading of TLS certificates
type certReloader struct {
	certFile string
	keyFile  string
	cert     *tls.Certificate
	mu       sync.RWMutex
}

// newCertReloader creates a new certificate reloader
func newCertReloader(certFile, keyFile string) (*certReloader, error) {
	cr := &certReloader{
		certFile: certFile,
		keyFile:  keyFile,
	}
	if err := cr.reload(); err != nil {
		return nil, err
	}
	return cr, nil
}

// reload loads the certificate from disk
func (cr *certReloader) reload() error {
	cert, err := tls.LoadX509KeyPair(cr.certFile, cr.keyFile)
	if err != nil {
		return err
	}
	cr.mu.Lock()
	cr.cert = &cert
	cr.mu.Unlock()
	log.Printf("TLS certificate reloaded from %s", cr.certFile)
	return nil
}

// getCertificate returns the current certificate (implements tls.Config.GetCertificate)
func (cr *certReloader) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.cert, nil
}

// startReloading starts periodic certificate reloading
func (cr *certReloader) startReloading(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cr.reload(); err != nil {
				log.Printf("ERROR: Failed to reload TLS certificate: %v", err)
			}
		case <-stopCh:
			log.Println("TLS certificate reloader stopped")
			return
		}
	}
}
