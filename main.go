package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/datum-cloud/galactic-agent/api/local"
	"github.com/datum-cloud/galactic-agent/api/remote"
)

var configFile string

func initConfig() {
	viper.SetDefault("srv6_net", "fc00::/56")
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

			_, err := EncodeVPCToSRv6Endpoint(viper.GetString("srv6_net"), "ffffffffffff", "ffff")
			if err != nil {
				log.Fatalf("srv6_endpoint invalid: %v", err)
			}

			l = local.Local{
				SocketPath: viper.GetString("socket_path"),
				RegisterHandler: func(vpc, vpcAttachment string, networks []string) error {
					srv6_endpoint, err := EncodeVPCToSRv6Endpoint(viper.GetString("srv6_net"), vpc, vpcAttachment)
					if err != nil {
						return err
					}
					for _, n := range networks {
						payload := fmt.Sprintf("REGISTER: network='%s', srv6_endpoint='%s'", n, srv6_endpoint)
						log.Println(payload)
						r.Send(payload)
					}
					return nil
				},
				DeregisterHandler: func(vpc, vpcAttachment string, networks []string) error {
					srv6_endpoint, err := EncodeVPCToSRv6Endpoint(viper.GetString("srv6_net"), vpc, vpcAttachment)
					if err != nil {
						return err
					}
					for _, n := range networks {
						payload := fmt.Sprintf("DEREGISTER: network='%s', srv6_endpoint='%s'", n, srv6_endpoint)
						log.Println(payload)
						r.Send(payload)
					}
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

func EncodeVPCToSRv6Endpoint(srv6_net, vpc, vpcAttachment string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(srv6_net)
	if err != nil {
		return "", err
	}
	if ip.To4() != nil {
		return "", fmt.Errorf("provided srv6_net is not IPv6: %s", srv6_net)
	}
	mask_len, _ := ipnet.Mask.Size()
	if mask_len > 64 {
		return "", fmt.Errorf("srv6_net must be at least 64 bits long")
	}

	vpcInt, err := strconv.ParseUint(vpc, 16, 64)
	if err != nil {
		return "", fmt.Errorf("invalid vpc %q: %w", vpc, err)
	}
	vpcAttachmentInt, err := strconv.ParseUint(vpcAttachment, 16, 16)
	if err != nil {
		return "", fmt.Errorf("invalid vpcAttachment %q: %w", vpcAttachment, err)
	}

	binary.BigEndian.PutUint64(ip[8:16], (vpcInt<<16)|vpcAttachmentInt)
	return ip.String(), nil
}
