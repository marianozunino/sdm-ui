/*
Copyright © 2024 Mariano Zunino <marianoz@posteo.net>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfgFile string

type conf struct {
	Email             string   `mapstructure:"email"`
	DBPath            string   `mapstructure:"dbPath"`
	Verbose           bool     `mapstructure:"verbose"`
	BalcklistPatterns []string `mapstructure:"blacklistPatterns"`
}

var confData conf = conf{
	Email:             "",
	DBPath:            "",
	Verbose:           false,
	BalcklistPatterns: []string{},
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sdm-ui",
	Short: "SDM UI - Wrapper for SDM CLI",
	Long: `
 ___ ___  __  __   _   _ ___
/ __|   \|  \/  | | | | |_ _|
\__ \ |) | |\/| | | |_| || |
|___/___/|_|  |_|  \___/|___| ` + VersionFromBuild() + `

SDM UI is a custom wrapper around StrongDM (SDM) designed to improve the developer experience (DX) on Linux.`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
		return initializeConfig(cmd)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", xdg.ConfigHome+"/sdm-ui.yaml", "config file")
	rootCmd.PersistentFlags().StringVarP(&confData.Email, "email", "e", "", "email address (overrides config file)")
	rootCmd.PersistentFlags().BoolVarP(&confData.Verbose, "verbose", "v", false, "verbose output (overrides config file)")
	rootCmd.PersistentFlags().StringVarP(&confData.DBPath, "db", "d", xdg.DataHome, "database path")

	rootCmd.MarkPersistentFlagRequired("email")
}

func initializeConfig(cmd *cobra.Command) error {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find configFolder directory.
		configFolder := cfgFile

		// Search config in home directory with name ".sdm-ui" (without extension).
		viper.AddConfigPath(configFolder)
		viper.SetConfigType("yaml")
		viper.SetConfigName("sdm-ui")
	}

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Bind the current command's flags to viper
	bindFlags(cmd)

	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && viper.IsSet(configName) {
			val := viper.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	// map blacklist patterns
	confData.BalcklistPatterns = viper.GetStringSlice("blacklistPatterns")
}
