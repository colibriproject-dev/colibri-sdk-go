package colibri

import (
	"context"
	"fmt"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/cloud"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/config"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/observer"
	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/validator"
)

const banner = `
      .   _            _ _ _          _ 
     { \/'o;===       | (_) |        (_)
.----'-/'-/  ___  ___ | |_| |__  _ __ _ 
 '-..-| /   / __ / _ \| | | '_ \| '__| |
    /\/\   | (__| (_) | | | |_) | |  | |
    '--'    \___ \___/|_|_|_.__/|_|  |_|
            project (%s)
`

// InitializeApp initializes all the components of the Colibri application.
// It loads the configuration, prints the banner and application name, and initializes
// the validator, observer, monitoring, and cloud services.
func InitializeApp() {
	if err := config.Load(); err != nil {
		logging.Fatal(context.Background()).Err(err).Msgf("an error on try load config")
	}

	printBanner()
	printApplicationName()

	logging.Initialize()
	validator.Initialize()
	observer.Initialize()
	monitoring.Initialize()
	cloud.Initialize()
}

func printBanner() {
	if config.IsDevelopmentEnvironment() {
		fmt.Printf(banner, config.VERSION)
	}
}

func printApplicationName() {
	fmt.Printf("\n# %s #\n\n", config.APP_NAME)
}
