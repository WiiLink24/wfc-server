package common

import (
	"encoding/xml"
	"os"
)

type Config struct {
	Username        string `xml:"username"`
	Password        string `xml:"password"`
	DatabaseAddress string `xml:"databaseAddress"`
	DatabaseName    string `xml:"databaseName"`

	DefaultAddress  string  `xml:"address"`
	GameSpyAddress  *string `xml:"gsAddress,omitempty"`
	NASAddress      *string `xml:"nasAddress,omitempty"`
	NASPort         string  `xml:"nasPort"`
	NASAddressHTTPS *string `xml:"nasAddressHttps,omitempty"`
	NASPortHTTPS    string  `xml:"nasPortHttps"`

	FrontendAddress        string `xml:"frontendAddress"`
	FrontendBackendAddress string `xml:"frontendBackendAddress"`
	BackendAddress         string `xml:"backendAddress"`
	BackendFrontendAddress string `xml:"backendFrontendAddress"`

	EnableHTTPS           bool  `xml:"enableHttps"`
	EnableHTTPSExploitWii *bool `xml:"enableHttpsExploitWii,omitempty"`
	EnableHTTPSExploitDS  *bool `xml:"enableHttpsExploitDS,omitempty"`

	LogLevel  *int   `xml:"logLevel"`
	LogOutput string `xml:"logOutput"`

	CertPath      string `xml:"certPath"`
	KeyPath       string `xml:"keyPath"`
	CertPathWii   string `xml:"certDerPathWii"`
	KeyPathWii    string `xml:"keyPathWii"`
	CertPathDS    string `xml:"certDerPathDS"`
	WiiCertPathDS string `xml:"wiiCertDerPathDS"`
	KeyPathDS     string `xml:"keyPathDS"`

	APISecret string `xml:"apiSecret"`

	AllowDefaultDolphinKeys bool `xml:"allowDefaultDolphinKeys"`

	ServerName string `xml:"serverName,omitempty"`
}

func GetConfig() Config {
	data, err := os.ReadFile("config.xml")
	if err != nil {
		panic(err)
	}

	var config Config
	config.AllowDefaultDolphinKeys = true
	config.ServerName = "Cosmos"

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

	if config.EnableHTTPSExploitWii == nil {
		enable := true
		config.EnableHTTPSExploitWii = &enable
	}

	if config.EnableHTTPSExploitDS == nil {
		enable := true
		config.EnableHTTPSExploitDS = &enable
	}

	if config.LogLevel == nil {
		level := 4
		config.LogLevel = &level
	}

	if config.LogOutput == "" {
		config.LogOutput = "StdOutAndFile"
	}

	if config.FrontendAddress == "" {
		config.FrontendAddress = "127.0.0.1:29998"
	}

	if config.BackendAddress == "" {
		config.BackendAddress = "127.0.0.1:29999"
	}

	if config.FrontendBackendAddress == "" {
		config.FrontendBackendAddress = config.BackendAddress
	}

	if config.BackendFrontendAddress == "" {
		config.BackendFrontendAddress = config.FrontendAddress
	}

	return config
}
