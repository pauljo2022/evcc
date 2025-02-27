package meter

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/openwb"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("openwb", NewOpenWBFromConfig)
}

// NewOpenWBFromConfig creates a new configurable meter
func NewOpenWBFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		mqtt.Config `mapstructure:",squash"`
		Topic       string
		Timeout     time.Duration
		Usage       string
	}{
		Topic:   "openWB",
		Timeout: 15 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger("openwb")

	client, err := mqtt.RegisteredClientOrDefault(log, cc.Config)
	if err != nil {
		return nil, err
	}

	// timeout handler
	to := provider.NewTimeoutHandler(provider.NewMqtt(log, client,
		fmt.Sprintf("%s/system/%s", cc.Topic, openwb.TimestampTopic), 1, cc.Timeout,
	).StringGetter())

	boolG := func(topic string) func() (bool, error) {
		g := provider.NewMqtt(log, client, topic, 1, 0).BoolGetter()
		return to.BoolGetter(g)
	}

	floatG := func(topic string) func() (float64, error) {
		g := provider.NewMqtt(log, client, topic, 1, 0).FloatGetter()
		return to.FloatGetter(g)
	}

	var power func() (float64, error)
	var soc func() (float64, error)
	var currents func() (float64, float64, float64, error)

	switch strings.ToLower(cc.Usage) {
	case "grid":
		power = floatG(fmt.Sprintf("%s/evu/%s", cc.Topic, openwb.PowerTopic))

		var curr []func() (float64, error)
		for i := 1; i <= 3; i++ {
			current := floatG(fmt.Sprintf("%s/evu/%s%d", cc.Topic, openwb.CurrentTopic, i))
			curr = append(curr, current)
		}

		currents = collectCurrentProviders(curr)

	case "pv":
		configuredG := boolG(fmt.Sprintf("%s/pv/1/%s", cc.Topic, openwb.PvConfigured)) // first pv
		configured, err := configuredG()
		if err != nil {
			return nil, err
		}

		if !configured {
			return nil, errors.New("pv not available")
		}

		power = floatG(fmt.Sprintf("%s/pv/%s", cc.Topic, openwb.PowerTopic))

	case "battery":
		configuredG := boolG(fmt.Sprintf("%s/housebattery/%s", cc.Topic, openwb.BatteryConfigured))
		configured, err := configuredG()
		if err != nil {
			return nil, err
		}

		if !configured {
			return nil, errors.New("battery not available")
		}

		power = floatG(fmt.Sprintf("%s/housebattery/%s", cc.Topic, openwb.PowerTopic))
		soc = floatG(fmt.Sprintf("%s/housebattery/%s", cc.Topic, openwb.SoCTopic))

	default:
		return nil, fmt.Errorf("invalid usage: %s", cc.Usage)
	}

	m, err := NewConfigurable(power)
	if err != nil {
		return nil, err
	}

	res := m.Decorate(nil, currents, soc)

	return res, nil
}
