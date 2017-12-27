package hm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type jsonRPCReq struct {
	Version string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  map[string]string `json:"params"`
}

type jsonRPCResp struct {
	Error  interface{}     `json:"error"`
	Result json.RawMessage `json:"result"`
}

const (
	jsonRPCVersion  = "1.1"
	jsonContentType = "application/json"
)

type Conn struct {
	hc         *http.Client
	ccuHost    string
	sessionKey string
}

func NewConnection(addr, user, pass string) (*Conn, error) {
	c := &Conn{
		hc:      http.DefaultClient,
		ccuHost: addr,
	}
	err := c.authenticate(user, pass)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Conn) authenticate(user, pass string) error {
	err := c.do("Session.login", map[string]string{
		"username": user,
		"password": pass,
	}, &c.sessionKey)

	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			var resp json.RawMessage
			err := c.do("Session.renew", map[string]string{"_session_id_": c.sessionKey}, &resp)
			if err != nil {
				log.Printf("session renewal failed: %v", err)
			}
		}
	}()
	return nil
}

func (c *Conn) do(method string, req map[string]string, resp interface{}) error {
	reqData := jsonRPCReq{
		Version: jsonRPCVersion,
		Method:  method,
		Params:  req,
	}
	if len(c.sessionKey) != 0 {
		req["_session_id_"] = c.sessionKey
	}
	buf, err := json.Marshal(&reqData)
	if err != nil {
		return err
	}
	httpResp, err := c.hc.Post(c.rpcURL(), jsonContentType, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	var rpcResponse jsonRPCResp
	err = json.NewDecoder(httpResp.Body).Decode(&rpcResponse)
	if err != nil {
		return err
	}
	err = json.Unmarshal(rpcResponse.Result, &resp)
	if err != nil {
		return err
	}
	if rpcResponse.Error != nil {
		return fmt.Errorf("CCU request failed: %v", rpcResponse.Error) // TODO: error could probably be reformatted
	}
	return nil
}

func (c *Conn) rpcURL() string {
	return "http://" + c.ccuHost + "/api/homematic.cgi"
}

func (c *Conn) Close() error {
	var m bool
	return c.do("Session.logout", map[string]string{}, &m)
}

type hmRoom struct {
	ID         string
	Name       string
	ChannelIDs []string `json:"channelIds"`
}

func (c *Conn) Rooms() ([]hmRoom, error) {
	var rooms []hmRoom
	err := c.do("Room.getAll", map[string]string{}, &rooms)
	return rooms, err
}

type hmDevice struct {
	ID       string
	Type     string
	Address  string
	Channels []hmChannel
}

type hmChannel struct {
	ID          string
	Address     string
	ChannelType string
}

func (c *Conn) Devices() ([]Device, error) {
	var hmDevs []hmDevice
	err := c.do("Device.listAllDetail", map[string]string{"interface": "BidCos-RF"}, &hmDevs)
	if err != nil {
		return nil, err
	}

	rooms, err := c.Rooms()
	if err != nil {
		log.Printf("could not retrieve room list: %v", err)
	}

	var devs []Device
	for _, hmDev := range hmDevs {
		for _, hmChan := range hmDev.Channels {
			var dev Device
			dev.HMAddress = hmChan.Address

			switch hmChan.ChannelType {
			case "SWITCH":
				dev.Type = DeviceTypeSwitch
				dev.SetValue = func(v int) error {
					var result bool
					req := map[string]string{
						"interface": "BidCos-RF",
						"address":   hmChan.Address,
						"valueKey":  "STATE",
						"type":      "string",
						"value":     strconv.Itoa(v),
					}
					return c.do("Interface.setValue", req, &result)
				}
				dev.Value = func() (int, error) {
					v, err := c.value(hmChan.Address, "STATE")
					if v == 0 {
						return 0, err
					}
					return 1, err
				}
			case "BLIND":
				dev.Type = DeviceTypeBlind
				dev.SetValue = func(v int) error {
					var result bool
					req := map[string]string{
						"interface": "BidCos-RF",
						"address":   hmChan.Address,
						"valueKey":  "LEVEL",
						"type":      "string",
						"value":     fmt.Sprintf("%v", v/100),
					}
					return c.do("Interface.setValue", req, &result)
				}
				dev.Value = func() (int, error) {
					v, err := c.value(hmChan.Address, "LEVEL")
					return int(v * 100), err
				}
			default:
				continue
			}
			dev.Room = c.associateRoom(rooms, hmChan)
			devs = append(devs, dev)
		}
	}
	return devs, nil
}

func (c *Conn) value(address, valType string) (float64, error) {
	req := map[string]string{
		"interface": "BidCos-RF",
		"address":   address,
		"valueKey":  valType,
	}
	var res string
	err := c.do("Interface.getValue", req, &res)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(res, 64)
}

func (c *Conn) associateRoom(rooms []hmRoom, ch hmChannel) string {
	for _, room := range rooms {
		for _, chanID := range room.ChannelIDs {
			if chanID == ch.ID {
				return room.Name
			}
		}
	}
	return ""
}
