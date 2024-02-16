package libsecret

import (
	libsecret "github.com/zalando/go-keyring"
)

const service_key = "sdm-credential"

type Keyring struct {
}

func (k *Keyring) GetSecret(email string) (string, error) {
	return libsecret.Get(service_key, email)
}

func (k *Keyring) SetSecret(email string, secret string) error {
	return libsecret.Set(service_key, email, secret)
}

func (k *Keyring) DeleteSecret(email string) error {
	return libsecret.Delete(service_key, email)
}
