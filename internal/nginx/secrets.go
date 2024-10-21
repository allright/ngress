package nginx

import (
	core "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type Secrets struct {
	secrets map[string]*Secret
}

func newSecrets() *Secrets {
	return &Secrets{
		secrets: make(map[string]*Secret),
	}
}

func (c *Secrets) add(secret *core.Secret) {
	s := newSecret(secret)
	c.secrets[s.name()] = s // overwrite if exist
	klog.Infof("added <SECRET:%v>(%v)", s.name(), s.string())
}

func (c *Secrets) remove(secret *core.Secret) {
	s := newSecret(secret)
	delete(c.secrets, s.name())
	klog.Infof("removed <SECRET:%v>(%v)", s.name(), s.string())
}

func (c *Secrets) get(secretName string) *Secret {
	secret, _ := c.secrets[secretName]
	return secret
}

func (c *Secrets) resetWriteMarks() {
	for _, secret := range c.secrets {
		secret.markForWrite(false)
	}
}

func (c *Secrets) fillCerts(certsDir string, certs map[string][]byte) {
	for _, secret := range c.secrets {
		secret.fillCerts(certsDir, certs)
	}
}
