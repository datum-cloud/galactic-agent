package remote

import (
	"context"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Remote struct {
	Host           string
	Port           int
	QoS            byte
	TopicRX        string
	TopicTX        string
	ReceiveHandler func([]byte) error

	client mqtt.Client
}

func (r *Remote) Run(ctx context.Context) error {
	mqtt_url := fmt.Sprintf("tcp://%s:%d", r.Host, r.Port)
	log.Printf("MQTT connecting: %s", mqtt_url)

	r.client = mqtt.NewClient(
		mqtt.NewClientOptions().
			AddBroker(mqtt_url),
	)
	if tok := r.client.Connect(); tok.Wait() && tok.Error() != nil {
		return tok.Error()
	}
	log.Println("MQTT connected")

	token := r.client.Subscribe(
		r.TopicRX,
		r.QoS,
		func(_ mqtt.Client, msg mqtt.Message) {
			payload := msg.Payload()
			if err := r.ReceiveHandler(payload); err != nil {
				log.Printf("MQTT ReceiveHandler failed for: %v", payload)
			}
		},
	)
	if !token.WaitTimeout(5*time.Second) || token.Error() != nil {
		return token.Error()
	}
	log.Printf("MQTT subscribed: %s", r.TopicRX)

	<-ctx.Done()
	if r.client.IsConnected() {
		r.client.Disconnect(250)
	}
	log.Println("MQTT disconnected")

	return nil
}

func (r *Remote) Send(payload interface{}) {
	token := r.client.Publish(r.TopicTX, r.QoS, false, payload)
	token.Wait()
}
