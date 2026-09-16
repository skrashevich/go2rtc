package hap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"sync"

	"github.com/AlexxIT/go2rtc/pkg/hap/tlv8"
)

// Character - Aqara props order
// Value should be omit for PW
// Value may be empty for PR
type Character struct {
	Desc string `json:"description,omitempty"`

	IID    uint64   `json:"iid"`
	Type   string   `json:"type"`
	Format string   `json:"format"`
	Value  any      `json:"value,omitempty"`
	Perms  []string `json:"perms"`

	//MaxLen   int    `json:"maxLen,omitempty"`
	//Unit     string `json:"unit,omitempty"`
	//MinValue any    `json:"minValue,omitempty"`
	//MaxValue any    `json:"maxValue,omitempty"`
	//MinStep  any    `json:"minStep,omitempty"`
	//ValidVal []any  `json:"valid-values,omitempty"`

	valueMu     sync.RWMutex
	listenersMu sync.Mutex
	listeners   map[io.Writer]bool
}

// GetValue and SetValue synchronize runtime characteristic updates.
func (c *Character) GetValue() any {
	c.valueMu.RLock()
	defer c.valueMu.RUnlock()
	return c.Value
}

func (c *Character) SetValue(value any) {
	c.valueMu.Lock()
	c.Value = value
	c.valueMu.Unlock()
}

func (c *Character) MarshalJSON() ([]byte, error) {
	c.valueMu.RLock()
	defer c.valueMu.RUnlock()
	type character Character
	return json.Marshal((*character)(c))
}

func (c *Character) AddListener(w io.Writer) {
	c.listenersMu.Lock()
	defer c.listenersMu.Unlock()
	if c.listeners == nil {
		c.listeners = map[io.Writer]bool{}
	}
	c.listeners[w] = true
}

func (c *Character) RemoveListener(w io.Writer) {
	c.listenersMu.Lock()
	defer c.listenersMu.Unlock()
	delete(c.listeners, w)

	if len(c.listeners) == 0 {
		c.listeners = nil
	}
}

func (c *Character) ListenerCount() int {
	c.listenersMu.Lock()
	defer c.listenersMu.Unlock()
	return len(c.listeners)
}

func (c *Character) NotifyListeners(ignore io.Writer) error {
	c.listenersMu.Lock()
	listeners := slices.Collect(maps.Keys(c.listeners))
	c.listenersMu.Unlock()
	if len(listeners) == 0 {
		return nil
	}

	data, err := c.GenerateEvent()
	if err != nil {
		return err
	}

	// Network writes must not hold the subscription lock.
	for _, w := range listeners {
		if w == ignore {
			continue
		}
		if _, err = w.Write(data); err != nil {
			// error not a problem - just remove listener
			c.RemoveListener(w)
		}
	}

	return nil
}

// GenerateEvent with raw HTTP headers
func (c *Character) GenerateEvent() (data []byte, err error) {
	v := JSONCharacters{
		Value: []JSONCharacter{
			{AID: DeviceAID, IID: c.IID, Value: c.GetValue()},
		},
	}
	if data, err = json.Marshal(v); err != nil {
		return
	}

	res := http.Response{
		StatusCode:    http.StatusOK,
		ProtoMajor:    1,
		ProtoMinor:    0,
		Header:        http.Header{"Content-Type": []string{MimeJSON}},
		ContentLength: int64(len(data)),
		Body:          io.NopCloser(bytes.NewReader(data)),
	}

	buf := bytes.NewBuffer([]byte{0})
	if err = res.Write(buf); err != nil {
		return
	}
	copy(buf.Bytes(), "EVENT")

	return buf.Bytes(), err
}

// Set new value and NotifyListeners
func (c *Character) Set(v any) (err error) {
	if err = c.Write(v); err != nil {
		return
	}
	return c.NotifyListeners(nil)
}

// Write new value with right format
func (c *Character) Write(v any) (err error) {
	switch c.Format {
	case "tlv8":
		var value string
		value, err = tlv8.MarshalBase64(v)
		if err == nil {
			c.SetValue(value)
		}

	case "bool":
		switch v := v.(type) {
		case bool:
			c.SetValue(v)
		case float64:
			c.SetValue(v != 0)
		}
	}
	return
}

// ReadTLV8 value to right struct
func (c *Character) ReadTLV8(v any) (err error) {
	if s, ok := c.GetValue().(string); ok {
		return tlv8.UnmarshalBase64(s, v)
	}
	return fmt.Errorf("hap: can't read value: %v", v)
}

func (c *Character) ReadBool() (bool, error) {
	if v, ok := c.GetValue().(bool); ok {
		return v, nil
	}
	return false, fmt.Errorf("hap: can't read value: %v", c.GetValue())
}

func (c *Character) String() string {
	data, err := json.Marshal(c)
	if err != nil {
		return "ERROR"
	}
	return string(data)
}
