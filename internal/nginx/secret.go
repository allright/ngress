package nginx

import (
	"fmt"
	"hash/adler32"
	core "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"path/filepath"
)

type Secret struct {
	secret    *core.Secret
	mustWrite bool
}

func newSecret(secret *core.Secret) *Secret {
	return &Secret{secret: secret}
}

func makeSecretName(namespace string, name string) string {
	return filepath.Join(namespace, name)
}

func (c *Secret) name() string {
	return makeSecretName(c.secret.Namespace, c.secret.Name)
}

func (c *Secret) path(certsDir string, name string) string {
	certificate := c.secret.Data[name]
	if certificate == nil {
		klog.Errorf("SECRET:%v file:%v NOT found", c.name(), name)
		return ""
	}
	return filepath.Join(certsDir, c.name(), name)
}

func (c *Secret) data(name string) []byte {
	return c.secret.Data[name]
}

func (c *Secret) string() string {
	data := ""
	for k, v := range c.secret.Data {
		if len(data) > 0 {
			data += ","
		}
		data += fmt.Sprintf("%v:%v:%08x", k, len(v), adler32.Checksum(v))
	}
	return data
}

func (c *Secret) markForWrite(mark bool) {
	c.mustWrite = mark
}

func (c *Secret) fillCerts(certsDir string, certs map[string][]byte) {
	if !c.mustWrite {
		return
	}
	for name, data := range c.secret.Data {
		certs[c.path(certsDir, name)] = data
	}
}
