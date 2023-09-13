package common

import (
	"encoding/xml"
	"io/ioutil"
)

type Config struct {
	Username        string `xml:"username"`
	Password        string `xml:"password"`
	DatabaseAddress string `xml:"databaseAddress"`
	DatabaseName    string `xml:"databaseName"`
	Address         string `xml:"address"`
}

func GetConfig() Config {
	data, err := ioutil.ReadFile("config.xml")
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
