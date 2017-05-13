// xright 2016 Mender Software A
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"

	"github.com/mendersoftware/deployments/config"
)

func main() {

	var configPath string
	var printVersion bool
	flag.StringVar(&configPath, "config", "", "Configuration file path. Supports JSON, TOML, YAML and HCL formatted configs.")
	flag.BoolVar(&printVersion, "version", false, "Show version")

	flag.Parse()

	if printVersion {
		fmt.Println("Version:", CreateVersionString())
		fmt.Println("BuildNumber:", BuildNumber)
		os.Exit(0)
	}

	l := log.New(log.Ctx{})

	configuration, err := HandleConfigFile(configPath)
	if err != nil {
		l.Fatalf("error loading configuration: %s", err)
	}

	dbSession, err := mgo.Dial(configuration.GetString(SettingMongo))
	if err != nil {
		l.Fatalf("failed to connect to DB: %v", err)
	}
	dbSession.SetSafe(&mgo.Safe{
		W: 1,
		J: true,
	})

	err = MigrateDb(context.Background(), DbVersion, nil, dbSession)
	if err != nil {
		l.Fatalf("failed to migrate DB data: %v", err)
	}

	l.Fatal(RunServer(configuration))
}

func HandleConfigFile(filePath string) (config.ConfigReader, error) {

	c := viper.New()

	// map settings such as foo.bar and foo-bar to FOO_BAR environment keys
	c.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Enable setting also other conig values by environment variables
	c.SetEnvPrefix("DEPLOYMENTS")
	c.AutomaticEnv()

	// Set default values for config
	SetDefaultConfigs(c)

	// Find and read the config file
	if filePath != "" {
		c.SetConfigFile(filePath)
		if err := c.ReadInConfig(); err != nil {
			return nil, err
		}
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
	config.SetDefault(SettingAwsS3Bucket, SettingAwsS3BucketDefault)
	config.SetDefault(SettingMongo, SettingMongoDefault)
	config.SetDefault(SettingGateway, SettingGatewayDefault)
}
