package tor

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"os"

	"github.com/cretz/bine/tor"
	"github.com/rs/zerolog/log"
)

var ports []net.Listener // list of ports

func Init() {

	// Start tor with default config (can set start conf's DebugWriter to os.Stdout for debug logs)
	log.Info().Msg("Starting and registering onion service, please wait a couple of minutes...")
	t, err := tor.Start(nil, &tor.StartConf{DebugWriter: os.Stdout, TorrcFile: "torrc"})
	if err != nil {
		return
	}

	// Wait at most a few minutes to publish the service
	listenCtx := context.Background()

	for _, port := range ports {
		onion, err := t.Listen(listenCtx, &tor.ListenConf{LocalListener: port, Version3: true})
		if err != nil {
			log.Error().Err(err).Msg("Unable to create onion service")
			return
		}
		PrKey, _ := SavePrivateKeyToString(onion.Key)
		log.Info().Str("onion", onion.ID+".onion").Int("port", onion.LocalListener.Addr().(*net.TCPAddr).Port).Str("key", PrKey).Msg("[tor] Service registered")
	}
}

func Listen(ln net.Listener) {
	ports = append(ports, ln)
	log.Trace().Int("port", ln.Addr().(*net.TCPAddr).Port).Msg("[tor] listen")
}

// SavePrivateKeyToString converts a private key to a PEM-formatted string
func SavePrivateKeyToString(privateKey crypto.PrivateKey) (string, error) {
	var block *pem.Block

	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		block = &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		}
	case *ecdsa.PrivateKey:
		bytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return "", err
		}
		block = &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: bytes,
		}
	default:
		return "", errors.New("unsupported key type")
	}

	return string(pem.EncodeToMemory(block)), nil
}

// LoadPrivateKeyFromString loads a private key from a PEM-formatted string
func LoadPrivateKeyFromString(pemString string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, errors.New("unsupported key type")
	}
}
