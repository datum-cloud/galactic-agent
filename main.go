package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/datum-cloud/galactic-agent/api/local"
	"github.com/datum-cloud/galactic-agent/api/remote"
)

var configFile string

func initConfig() {
	viper.SetDefault("socket_path", "/var/run/galactic/agent.sock")
	viper.SetDefault("mqtt_host", "mqtt")
	viper.SetDefault("mqtt_port", 1883)
	viper.SetDefault("mqtt_qos", 0)
	viper.SetDefault("mqtt_topic_receive", "galactic/default/receive")
	viper.SetDefault("mqtt_topic_send", "galactic/default/send")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		log.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		log.Printf("No config file found - using defaults.")
	}
}

var (
	l local.Local
	r remote.Remote
)

func main() {
	cmd := &cobra.Command{
		Use:   "galactic-agent",
		Short: "Galactic Agent",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig()
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop() //nolint:errcheck

			l = local.Local{
				SocketPath: viper.GetString("socket_path"),
				RegisterHandler: func(vpc, vpcAttachment string, networks []string) error {
					log.Printf("REGISTER:   vpc='%v', vpcattachment='%v', networks='%v'\n", vpc, vpcAttachment, networks)
					r.Send(fmt.Sprintf("REGISTER:   vpc='%v', vpcattachment='%v', networks='%v'", vpc, vpcAttachment, networks))
					return nil
				},
				DeregisterHandler: func(vpc, vpcAttachment string, networks []string) error {
					log.Printf("DEREGISTER: vpc='%v', vpcattachment='%v', networks='%v'\n", vpc, vpcAttachment, networks)
					r.Send(fmt.Sprintf("DEREGISTER: vpc='%v', vpcattachment='%v', networks='%v'", vpc, vpcAttachment, networks))
					return nil
				},
			}

			r = remote.Remote{
				Host:    viper.GetString("mqtt_host"),
				Port:    viper.GetInt("mqtt_port"),
				QoS:     byte(viper.GetInt("mqtt_qos")),
				TopicRX: viper.GetString("mqtt_topic_receive"),
				TopicTX: viper.GetString("mqtt_topic_send"),
				ReceiveHandler: func(payload interface{}) {
					log.Printf("MQTT received: %s", payload)
				},
			}

			g, ctx := errgroup.WithContext(ctx)
			g.Go(func() error {
				return l.Serve(ctx)
			})
			g.Go(func() error {
				return r.Run(ctx)
			})
			if err := g.Wait(); err != nil {
				log.Printf("Error: %v", err)
			}
			log.Printf("Shutdown")
		},
	}
	cmd.PersistentFlags().StringVar(&configFile, "config", "", "config file")
	cmd.SetArgs(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}
