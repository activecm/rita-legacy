package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/activecm/mgosec"
	"github.com/blang/semver"
)

type (
	//RunningCfg holds configuration options that are parsed at run time
	RunningCfg struct {
		MongoDB MongoDBRunningCfg
		Version semver.Version
	}

	//MongoDBRunningCfg holds parsed information for connecting to MongoDB
	MongoDBRunningCfg struct {
		AuthMechanismParsed mgosec.AuthMechanism
		TLS                 struct {
			TLSConfig *tls.Config
		}
	}
)

// loadRunningConfig attempts deserializes data in the static config
func loadRunningConfig(config *StaticCfg) (*RunningCfg, error) {
	var outConfig = new(RunningCfg)
	var err error

	//parse the tls configuration
	if config.MongoDB.TLS.Enabled {
		tlsConf := &tls.Config{}
		if !config.MongoDB.TLS.VerifyCertificate {
			tlsConf.InsecureSkipVerify = true
		}
		if len(config.MongoDB.TLS.CAFile) > 0 {
			pem, err2 := ioutil.ReadFile(config.MongoDB.TLS.CAFile)
			err = err2
			if err != nil {
				fmt.Println("[!] Could not read MongoDB CA file")
			} else {
				tlsConf.RootCAs = x509.NewCertPool()
				tlsConf.RootCAs.AppendCertsFromPEM(pem)
			}
		}
		outConfig.MongoDB.TLS.TLSConfig = tlsConf
	}

	//parse out the mongo authentication mechanism
	authMechanism, err := mgosec.ParseAuthMechanism(
		config.MongoDB.AuthMechanism,
	)
	if err != nil {
		authMechanism = mgosec.None
		fmt.Println("[!] Could not parse MongoDB authentication mechanism")
	}
	outConfig.MongoDB.AuthMechanismParsed = authMechanism

	outConfig.Version, err = semver.ParseTolerant(config.Version)
	return outConfig, err
}
