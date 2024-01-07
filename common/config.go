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
	Address         string `xml:"address"`
	Port            string `xml:"nasPort"`
	PortHTTPS       string `xml:"nasPortHttps"`
	EnableHTTPS     bool   `xml:"enableHttps"`
	LogLevel        int    `xml:"logLevel"`
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

	return config
}
