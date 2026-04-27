package auth

import "github.com/zalando/go-keyring"

const serviceName = "beanctl"

func key(profile string) string {
	return "beanctl:" + profile
}

func SaveRaw(profile, value string) error {
	return keyring.Set(serviceName, key(profile), value)
}

func LoadRaw(profile string) (string, error) {
	return keyring.Get(serviceName, key(profile))
}

func Delete(profile string) error {
	return keyring.Delete(serviceName, key(profile))
}
