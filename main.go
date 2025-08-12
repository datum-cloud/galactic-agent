package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/datum-cloud/galactic-agent/api/local"
)

var configFile string

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}
	viper.AutomaticEnv()
	viper.SetDefault("socket_path", "/var/run/galactic/agent.sock")
	if err := viper.ReadInConfig(); err == nil {
		log.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		log.Printf("No config file found - using defaults.")
	}
}

func main() {
	cmd := &cobra.Command{
		Use:   "galactic-agent",
		Short: "Galactic Agent",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := local.Serve(viper.GetString("socket_path")); err != nil {
				log.Fatalf("Serve failed: %v", err)
			}
		},
	}
	cmd.PersistentFlags().StringVar(&configFile, "config", "", "config file")
	cmd.SetArgs(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}
