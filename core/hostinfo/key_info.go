package hostinfo

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"

	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/pkg/errors"
	"github.com/rancher/log"
)

type KeyCollector struct {
	key string
}

func (k KeyCollector) GetData() (map[string]interface{}, error) {
	key, err := k.getKey()
	return map[string]interface{}{
		"data": key,
	}, err
}

func (k KeyCollector) getKey() (string, error) {
	if k.key != "" {
		return k.key, nil
	}

	fileName := config.KeyFile()
	root, leaf, err := managedKeyLocation(fileName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return "", err
	}
	defer directory.Close()
	var bytes []byte
	keyFile, err := directory.Open(leaf)
	if err == nil {
		bytes, err = io.ReadAll(io.LimitReader(keyFile, maxHostKeySize+1))
		closeErr := keyFile.Close()
		if err == nil {
			err = closeErr
		}
		if len(bytes) > maxHostKeySize {
			return "", errors.New("host key exceeds size limit")
		}
	}
	if os.IsNotExist(err) {
		bytes, err = k.genKey()
		if err != nil {
			return "", err
		}
		keyFile, err = directory.OpenFile(leaf, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
		if err != nil {
			return "", err
		}
		if _, err = keyFile.Write(bytes); err != nil {
			_ = keyFile.Close()
			return "", err
		}
		if err = keyFile.Close(); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	b, _ := pem.Decode(bytes)
	key, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		return "", err
	}

	bytes, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal public key")
	}

	bytes = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: bytes,
	})
	k.key = string(bytes)

	return k.key, nil
}

const maxHostKeySize = 64 * 1024

func managedKeyLocation(fileName string) (string, string, error) {
	cleaned := filepath.Clean(fileName)
	if filepath.Base(cleaned) != "host.key" {
		return "", "", errors.New("host key path is not approved")
	}
	return filepath.Dir(cleaned), "host.key", nil
}

func (k KeyCollector) genKey() ([]byte, error) {
	log.Info("Generating host key")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	log.Info("Done generating host key")
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), nil
}

func (k KeyCollector) KeyName() string {
	return "hostKey"
}

func (k KeyCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
