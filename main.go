package main

import (
	"flag"
	"log"

	"github.com/mendersoftware/artifacts/config"
	"github.com/spf13/viper"
)

func main() {

	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "Configuration file path. Supports JSON, TOML, YAML and HCL formatted configs.")

	flag.Parse()

	c := viper.New()
	c.SetConfigFile(configPath)

	// Set default values for config
	SetDefaultConfigs(c)

	// Find and read the config file
	if err := c.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}

	// Validate config
	if err := config.ValidateConfig(c,
		ValidateAwsAuth,
		ValidateHttps,
	); err != nil {
		log.Fatalln(err)
	}

	log.Fatalln(RunServer(c))
}

func SetDefaultConfigs(config *viper.Viper) {
	config.SetDefault(SettingListen, SettingListenDefault)
	config.SetDefault(SettingAwsS3Region, SettingAwsS3RegionDefault)
	config.SetDefault(SettingAweS3Bucket, SettingAwsS3BucketDefault)
}
