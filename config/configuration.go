package config

import "errors"

type ServiceConfig struct {
	ChirpStack ChirpStackConfig
}

type ChirpStackConfig struct {
	Version     string
	Host        string
	Username    string
	Password    string
	ActivateKey string
}

func (sw *ServiceConfig) UpdateFromRaw(rawConfig interface{}) bool {
	configuration, ok := rawConfig.(*ServiceConfig)
	if !ok {
		return false
	}

	*sw = *configuration

	return true
}

func (scc *ChirpStackConfig) Validate() error {
	if len(scc.Version) == 0 {
		return errors.New("ChirpStack.Version configuration setting can not be blank")
	}

	if len(scc.Host) == 0 {
		return errors.New("ChirpStack.Host configuration setting can not be blank")
	}

	if len(scc.Username) == 0 {
		return errors.New("ChirpStack.Username configuration setting can not be blank")
	}

	if len(scc.Password) == 0 {
		return errors.New("ChirpStack.Password configuration setting can not be blank")
	}

	if len(scc.ActivateKey) == 0 {
		return errors.New("ChirpStack.ActivateKey configuration setting can not be blank")
	}

	return nil
}
