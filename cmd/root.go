/*
Copyright © 2025 Mariano Zunino <marianoz@posteo.net>

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
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Configuration structure
type config struct {
	Email             string   `mapstructure:"email"`
	DBPath            string   `mapstructure:"dbPath"`
	Verbose           bool     `mapstructure:"verbose"`
	BlacklistPatterns []string `mapstructure:"blacklistPatterns"`
}

// Global configuration instance
var (
	cfgFile  string
	confData = config{
		Email:             "",
		DBPath:            xdg.DataHome,
		Verbose:           false,
		BlacklistPatterns: []string{},
	}
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sdm-ui",
	Short: "SDM UI - Wrapper for SDM CLI",
	Long: `
 ___ ___  __  __   _   _ ___
/ __|   \|  \/  | | | | |_ _|
\__ \ |) | |\/| | | |_| || |
|___/___/|_|  |_|  \___/|___| ` + VersionFromBuild() + `

SDM UI is a custom wrapper around StrongDM (SDM) designed to improve
the developer experience (DX) on Linux.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadConfig(cmd)
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// init sets up flags and configuration
func init() {
	defaultConfigPath := filepath.Join(xdg.ConfigHome, "sdm-ui.yaml")

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultConfigPath, "config file path")
	rootCmd.PersistentFlags().StringVarP(&confData.Email, "email", "e", "", "email address")
	rootCmd.PersistentFlags().BoolVarP(&confData.Verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&confData.DBPath, "db", "d", xdg.DataHome, "database path")

	rootCmd.MarkPersistentFlagRequired("email")
}

// loadConfig loads configuration from file and environment
func loadConfig(cmd *cobra.Command) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(filepath.Dir(cfgFile))
		viper.SetConfigType("yaml")
		viper.SetConfigName("sdm-ui")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("could not read config file: %w", err)
		}
	}

	cmd.Flags().Visit(func(f *pflag.Flag) {
		viper.Set(f.Name, f.Value.String())
	})

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	confData.BlacklistPatterns = viper.GetStringSlice("blacklistPatterns")

	return nil
}
