package config

import (
	"encoding/json"
	"os"
)

var Ip = Config{
	Pool: []string{},
}

type Config struct {
	Pool []string
}

func init() {
	f, err := os.Open("ips.json")

	if err != nil {
		panic(err)
	}

	err = json.NewDecoder(f).Decode(&Ip)

	if err != nil {
		panic(err)
	}
}
