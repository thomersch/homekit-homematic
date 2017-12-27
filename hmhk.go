package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/thomersch/homematic-homekit/hm"

	"github.com/brutella/hc"
	hcacc "github.com/brutella/hc/accessory"
	hclog "github.com/brutella/hc/log"
	"github.com/brutella/hc/service"
)

var (
	// refreshInterval defines how often sensors will be asked for current state.
	refreshInterval = time.Duration(1 * time.Minute)
	// gracePeriod defines after how long an action is performed, sensors are polled.
	gracePeriod = time.Duration(5 * time.Second)
)

func main() {
	hmCCUAddress := os.Getenv("HM_CCU_ADDRESS")
	if len(hmCCUAddress) == 0 {
		log.Fatal("Please specify address of CCU in 'HM_CCU_ADDRESS' enviornment variable.")
	}
	hmCCUUser := os.Getenv("HM_CCU_USER")
	if len(hmCCUUser) == 0 {
		hmCCUUser = "Admin"
	}
	hmCCUPassword := os.Getenv("HM_CCU_PASSWORD")

	log.Println("Opening connection to CCU...")
	conn, err := hm.NewConnection(hmCCUAddress, hmCCUUser, hmCCUPassword)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Retrieving devices from hub, this may take a moment.")
	devs, err := conn.Devices()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Found %v devices.", len(devs))

	var hcAccs []*hcacc.Accessory
	for num, dev := range devs {
		var acc hcacc.Accessory
		info := hcacc.Info{
			Name:         "dev" + strconv.Itoa(num),
			SerialNumber: dev.HMAddress,
			Manufacturer: "Homematic",
		}

		switch dev.Type {
		case hm.DeviceTypeSwitch:
			acc = newSwitch(info, dev)
		case hm.DeviceTypeBlind:
			acc = newBlind(info, dev)
		default:
			log.Printf("unmappable device class: %v", dev.Type)
			continue
		}
		hcAccs = append(hcAccs, &acc)
	}

	hclog.Info.Disable() // hc is too chatty, let's disable its logging
	hcCfg := hc.Config{Pin: "00102003"}
	hcTransport, err := hc.NewIPTransport(hcCfg, hcacc.New(hcacc.Info{
		Name:         "HomematicCCU",
		SerialNumber: "1",
		Manufacturer: "Homematic feat. Thomas Skowron",
		Model:        "CCU",
	}, hcacc.TypeBridge), hcAccs...)
	if err != nil {
		log.Fatal(err)
	}

	hc.OnTermination(func() {
		log.Println("Closing session...")
		err := conn.Close()
		if err != nil {
			log.Printf("session not closed cleanly: %v", err)
		}
		<-hcTransport.Stop()
	})
	log.Println("System launch complete.")
	hcTransport.Start()
}

func newBlind(info hcacc.Info, dev hm.Device) hcacc.Accessory {
	acc := *hcacc.New(info, hcacc.TypeWindowCovering)
	windowCov := service.NewWindowCovering()

	tc := hm.NewTicker(refreshInterval)
	tc <- time.Now() // to get initial value asap

	windowCov.TargetPosition.OnValueRemoteUpdate(func(val int) {
		log.Printf("Setting value of %v to %v", dev, val)
		dev.SetValue(val)
		windowCov.CurrentPosition.SetValue(val)

		time.AfterFunc(3*gracePeriod, func() { log.Println("Refreshing value"); tc <- time.Now() })
	})

	go func() {
		for range tc {
			log.Printf("Refreshing %v", dev)
			val, err := dev.Value()
			if err != nil {
				log.Printf("could not retrieve current value for %v: %v", dev, err)
			} else {
				windowCov.CurrentPosition.SetValue(val)
			}
		}
	}()

	// Set Target position to current value, otherwise blind may show up as "Closing..." until an action is performed.
	val, err := dev.Value()
	if err == nil {
		windowCov.TargetPosition.SetValue(val)
	}

	acc.AddService(windowCov.Service)
	return acc
}

func newSwitch(info hcacc.Info, dev hm.Device) hcacc.Accessory {
	acc := *hcacc.New(info, hcacc.TypeSwitch)
	sw := service.NewSwitch()

	tc := hm.NewTicker(refreshInterval)
	tc <- time.Now() // to get initial value asap

	sw.On.OnValueRemoteUpdate(func(on bool) {
		log.Printf("Setting value of %v to %v", dev, on)
		if on {
			dev.SetValue(1)
		} else {
			dev.SetValue(0)
		}
		time.AfterFunc(gracePeriod, func() { tc <- time.Now() })
	})

	go func() {
		for range tc {
			log.Printf("Refreshing %v", dev)
			val, err := dev.Value()
			if err != nil {
				log.Printf("could not retrieve current value for %v: %v", dev, err)
			} else {
				if val == 0 {
					sw.On.SetValue(false)
				} else {
					sw.On.SetValue(true)
				}
			}
		}
	}()
	acc.AddService(sw.Service)
	return acc
}
