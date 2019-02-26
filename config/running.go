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

// initRunningConfig uses data in the static config initialize
// the passed in running config
func initRunningConfig(static *StaticCfg, running *RunningCfg) error {
	var err error

	//parse the tls configuration
	if static.MongoDB.TLS.Enabled {
		tlsConf := &tls.Config{}
		if !static.MongoDB.TLS.VerifyCertificate {
			tlsConf.InsecureSkipVerify = true
		}
		if len(static.MongoDB.TLS.CAFile) > 0 {
			pem, err2 := ioutil.ReadFile(static.MongoDB.TLS.CAFile)
			err = err2
			if err != nil {
				fmt.Println("[!] Could not read MongoDB CA file")
			} else {
				tlsConf.RootCAs = x509.NewCertPool()
				tlsConf.RootCAs.AppendCertsFromPEM(pem)
			}
		}
		running.MongoDB.TLS.TLSConfig = tlsConf
	}

	//parse out the mongo authentication mechanism
	authMechanism, err := mgosec.ParseAuthMechanism(
		static.MongoDB.AuthMechanism,
	)
	if err != nil {
		authMechanism = mgosec.None
		fmt.Println("[!] Could not parse MongoDB authentication mechanism")
	}
	running.MongoDB.AuthMechanismParsed = authMechanism

	running.Version, err = semver.ParseTolerant(static.Version)
	if err != nil {
		fmt.Println("\t[!] Version error: please ensure that you cloned the git repo and are using make to build.")
		fmt.Println("\t[!] See the following resources for further information:")
		fmt.Println("\t[>] https://github.com/activecm/rita/blob/master/Contributing.md#common-issues")
		fmt.Println("\t[>] https://github.com/activecm/rita/blob/master/docs/Manual%20Installation.md")
	}
	return err
}
