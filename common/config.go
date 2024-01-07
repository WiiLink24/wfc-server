package common

import (
	"encoding/xml"
	"os"
)

type Config struct {
	Username           string  `xml:"username"`
	Password           string  `xml:"password"`
	DatabaseAddress    string  `xml:"databaseAddress"`
	DatabaseName       string  `xml:"databaseName"`
	DefaultAddress     string  `xml:"address"`
	GameSpyAddress     *string `xml:"gsAddress,omitempty"`
	NASAddress         *string `xml:"nasAddress,omitempty"`
	NASPort            string  `xml:"nasPort"`
	NASAddressHTTPS    *string `xml:"nasAddressHttps,omitempty"`
	NASPortHTTPS       string  `xml:"nasPortHttps"`
	EnableHTTPS        bool    `xml:"enableHttps"`
	EnableHTTPSExploit *bool   `xml:"enableHttpsExploit,omitempty"`
	LogLevel           *int    `xml:"logLevel"`
	CertPath           string  `xml:"certPath"`
	KeyPath            string  `xml:"keyPath"`
	CertPathWii        string  `xml:"certDerPathWii"`
	KeyPathWii         string  `xml:"keyPathWii"`
}

func GetConfig() Config {
	data, err := os.ReadFile("config.xml")
	if err != nil {
		panic(err)
	}

	var config Config
	err = xml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	if config.GameSpyAddress == nil {
		config.GameSpyAddress = &config.DefaultAddress
	}

	if config.NASAddress == nil {
		config.NASAddress = &config.DefaultAddress
	}

	if config.NASAddressHTTPS == nil {
		if config.NASAddress != nil {
			config.NASAddressHTTPS = config.NASAddress
		} else {
			config.NASAddressHTTPS = &config.DefaultAddress
		}
	}

	if config.EnableHTTPSExploit == nil {
		enable := true
		config.EnableHTTPSExploit = &enable
	}

	if config.LogLevel == nil {
		level := 4
		config.LogLevel = &level
	}

	return config
}
