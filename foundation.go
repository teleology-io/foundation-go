package foundation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/teleology-io/hermes"
)

type Foundation struct {
	url         string
	apiKey      string
	uid         *string
	config      interface{}
	environment interface{}
	variables   map[string]map[string]interface{}
	callback    func(event string, data interface{}, err error)
	socketURL   string
	client      hermes.Client
	conn        *websocket.Conn
}

func Str(value string) *string {
	return &value
}

func New(url string, apiKey string, uid *string) *Foundation {
	f := &Foundation{
		url:       url,
		apiKey:    apiKey,
		uid:       uid,
		variables: make(map[string]map[string]interface{}),
		socketURL: strings.Replace(fmt.Sprintf("%s%s?apiKey=%s", url, "/v1/realtime", apiKey), "http", "ws", -1),
		client: hermes.Create(hermes.ClientConfiguration{
			BaseURL: url,
			Headers: hermes.Headers{
				"X-Api-Key": apiKey,
			},
		}),
	}

	go f.realtime()

	return f
}

func (f *Foundation) realtime() {
	for {
		u, _ := url.Parse(f.socketURL)
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			return
		}
		f.conn = c

		for {
			_, msg, err := f.conn.ReadMessage()
			if err != nil {
				break
			}
			f.handleMessage(msg)
		}

		f.conn.Close()
	}
}

func (f *Foundation) GetEnvironment() (interface{}, error) {
	if f.environment == nil {
		resp, err := f.client.Send(hermes.Request{
			Url: "/v1/environment",
		})
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(string(resp.Data))
		}

		if err := json.Unmarshal(resp.Data, &f.environment); err != nil {
			return nil, err
		}
	}

	return f.environment, nil
}

func (f *Foundation) GetConfiguration() (interface{}, error) {
	if f.config == nil {
		resp, err := f.client.Send(hermes.Request{
			Url: "/v1/configuration",
		})
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf(string(resp.Data))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return nil, err
		}

		if content, ok := result["content"].(string); ok {
			var contentData interface{}
			if err := json.Unmarshal([]byte(content), &contentData); err == nil {
				f.config = contentData
			} else {
				f.config = content
			}
		}
	}

	return f.config, nil
}

func (f *Foundation) GetVariable(name string, uid *string, fallback interface{}) (interface{}, error) {
	if value, exists := f.variables[name]; exists && len(value) != 0 {
		return value["value"], nil
	}

	data := map[string]string{
		"name": name,
	}
	if f.uid != nil {
		data["uid"] = *f.uid
	}
	if uid != nil {
		data["uid"] = *uid
	}
	resp, err := f.client.Send(hermes.Request{
		Method: hermes.POST,
		Url:    "/v1/variable",
		Headers: hermes.Headers{
			"Content-Type": "application/json",
		},
		Data: data,
	})
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return nil, err
		}
		f.variables[name] = result
		return result["value"], nil
	} else if resp.StatusCode == http.StatusNotFound {
		return fallback, nil
	} else {
		return nil, fmt.Errorf(string(resp.Data))
	}
}

func (f *Foundation) Subscribe(callback func(event string, data interface{}, err error)) {
	f.callback = callback
}

func (f *Foundation) handleMessage(message []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		return
	}

	event, _ := data["type"].(string)
	switch event {
	case "variable.updated":
		name, _ := data["payload"].(map[string]interface{})["name"].(string)
		f.variables[name] = nil
		_, err := f.GetVariable(name, nil, nil)
		if f.callback != nil {
			f.callback(event, f.variables[name], err)
		}
	case "configuration.published":
		f.config = nil
		if f.callback != nil {
			data, err := f.GetConfiguration()
			f.callback(event, data, err)
		}
	case "environment.updated":
		f.environment = nil
		if f.callback != nil {
			data, err := f.GetConfiguration()
			f.callback(event, data, err)
		}
	}
}
