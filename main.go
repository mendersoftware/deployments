package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mendersoftware/artifacts/config"
	"github.com/spf13/viper"
)

func main() {

	var configPath string
	var printVersion bool
	flag.StringVar(&configPath, "config", "config.yaml", "Configuration file path. Supports JSON, TOML, YAML and HCL formatted configs.")
	flag.BoolVar(&printVersion, "version", false, "Show version")

	flag.Parse()

	if printVersion {
		fmt.Println("Version:", CreateVersionString())
		fmt.Println("BuildNumber:", BuildNumber)
		os.Exit(0)
	}

	configuration, err := HandleConfigFile(configPath)
	if err != nil {
		log.Fatalln(err)
	}

	log.Fatalln(RunServer(configuration))
}

func HandleConfigFile(filePath string) (config.ConfigReader, error) {

	c := viper.New()
	c.SetConfigFile(filePath)

	// Set default values for config
	SetDefaultConfigs(c)

	// Find and read the config file
	if err := c.ReadInConfig(); err != nil {
		return nil, err
	}

	// Validate config
	if err := config.ValidateConfig(c,
		ValidateAwsAuth,
		ValidateHttps,
	); err != nil {
		return nil, err
	}

	return c, nil
}

func SetDefaultConfigs(config *viper.Viper) {
	config.SetDefault(SettingListen, SettingListenDefault)
	config.SetDefault(SettingAwsS3Region, SettingAwsS3RegionDefault)
	config.SetDefault(SettingAweS3Bucket, SettingAwsS3BucketDefault)
}
