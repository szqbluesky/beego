// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"

	"github.com/astaxie/beego/pkg/infrastructure/config"
	"github.com/astaxie/beego/pkg/infrastructure/logs"
)

// JSONConfig is a json config parser and implements Config interface.
type JSONConfig struct {
}

// Parse returns a ConfigContainer with parsed json config map.
func (js *JSONConfig) Parse(filename string) (config.Configer, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return js.ParseData(content)
}

// ParseData returns a ConfigContainer with json string
func (js *JSONConfig) ParseData(data []byte) (config.Configer, error) {
	x := &JSONConfigContainer{
		data: make(map[string]interface{}),
	}
	err := json.Unmarshal(data, &x.data)
	if err != nil {
		var wrappingArray []interface{}
		err2 := json.Unmarshal(data, &wrappingArray)
		if err2 != nil {
			return nil, err
		}
		x.data["rootArray"] = wrappingArray
	}

	x.data = config.ExpandValueEnvForMap(x.data)

	return x, nil
}

// JSONConfigContainer is a config which represents the json configuration.
// Only when get value, support key as section:name type.
type JSONConfigContainer struct {
	data map[string]interface{}
	sync.RWMutex
}

func (c *JSONConfigContainer) Unmarshaler(ctx context.Context, prefix string, obj interface{}, opt ...config.DecodeOption) error {
	sub, err := c.sub(ctx, prefix)
	if err != nil {
		return err
	}
	return mapstructure.Decode(sub, obj)
}

func (c *JSONConfigContainer) Sub(ctx context.Context, key string) (config.Configer, error) {
	sub, err := c.sub(ctx, key)
	if err != nil {
		return nil, err
	}
	return &JSONConfigContainer{
		data: sub,
	}, nil
}

func (c *JSONConfigContainer) sub(ctx context.Context, key string) (map[string]interface{}, error) {
	if key == "" {
		return c.data, nil
	}
	value, ok := c.data[key]
	if !ok {
		return nil, errors.New(fmt.Sprintf("key is not found: %s", key))
	}

	res, ok := value.(map[string]interface{})
	if !ok {
		return nil, errors.New(fmt.Sprintf("the type of value is invalid, key: %s", key))
	}
	return res, nil
}

func (c *JSONConfigContainer) OnChange(ctx context.Context, key string, fn func(value string)) {
	logs.Warn("unsupported operation")
}

// Bool returns the boolean value for a given key.
func (c *JSONConfigContainer) Bool(ctx context.Context, key string) (bool, error) {
	val := c.getData(key)
	if val != nil {
		return config.ParseBool(val)
	}
	return false, fmt.Errorf("not exist key: %q", key)
}

// DefaultBool return the bool value if has no error
// otherwise return the defaultval
func (c *JSONConfigContainer) DefaultBool(ctx context.Context, key string, defaultVal bool) bool {
	if v, err := c.Bool(ctx, key); err == nil {
		return v
	}
	return defaultVal
}

// Int returns the integer value for a given key.
func (c *JSONConfigContainer) Int(ctx context.Context, key string) (int, error) {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(float64); ok {
			return int(v), nil
		} else if v, ok := val.(string); ok {
			return strconv.Atoi(v)
		}
		return 0, errors.New("not valid value")
	}
	return 0, errors.New("not exist key:" + key)
}

// DefaultInt returns the integer value for a given key.
// if err != nil return defaultval
func (c *JSONConfigContainer) DefaultInt(ctx context.Context, key string, defaultVal int) int {
	if v, err := c.Int(ctx, key); err == nil {
		return v
	}
	return defaultVal
}

// Int64 returns the int64 value for a given key.
func (c *JSONConfigContainer) Int64(ctx context.Context, key string) (int64, error) {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(float64); ok {
			return int64(v), nil
		}
		return 0, errors.New("not int64 value")
	}
	return 0, errors.New("not exist key:" + key)
}

// DefaultInt64 returns the int64 value for a given key.
// if err != nil return defaultval
func (c *JSONConfigContainer) DefaultInt64(ctx context.Context, key string, defaultVal int64) int64 {
	if v, err := c.Int64(ctx, key); err == nil {
		return v
	}
	return defaultVal
}

// Float returns the float value for a given key.
func (c *JSONConfigContainer) Float(ctx context.Context, key string) (float64, error) {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(float64); ok {
			return v, nil
		}
		return 0.0, errors.New("not float64 value")
	}
	return 0.0, errors.New("not exist key:" + key)
}

// DefaultFloat returns the float64 value for a given key.
// if err != nil return defaultval
func (c *JSONConfigContainer) DefaultFloat(ctx context.Context, key string, defaultVal float64) float64 {
	if v, err := c.Float(ctx, key); err == nil {
		return v
	}
	return defaultVal
}

// String returns the string value for a given key.
func (c *JSONConfigContainer) String(ctx context.Context, key string) (string, error) {
	val := c.getData(key)
	if val != nil {
		if v, ok := val.(string); ok {
			return v, nil
		}
	}
	return "", nil
}

// DefaultString returns the string value for a given key.
// if err != nil return defaultval
func (c *JSONConfigContainer) DefaultString(ctx context.Context, key string, defaultVal string) string {
	// TODO FIXME should not use "" to replace non existence
	if v, err := c.String(ctx, key); v != "" && err == nil {
		return v
	}
	return defaultVal
}

// Strings returns the []string value for a given key.
func (c *JSONConfigContainer) Strings(ctx context.Context, key string) ([]string, error) {
	stringVal, err := c.String(nil, key)
	if stringVal == "" || err != nil {
		return nil, err
	}
	return strings.Split(stringVal, ";"), nil
}

// DefaultStrings returns the []string value for a given key.
// if err != nil return defaultval
func (c *JSONConfigContainer) DefaultStrings(ctx context.Context, key string, defaultVal []string) []string {
	if v, err := c.Strings(ctx, key); v != nil && err == nil {
		return v
	}
	return defaultVal
}

// GetSection returns map for the given section
func (c *JSONConfigContainer) GetSection(ctx context.Context, section string) (map[string]string, error) {
	if v, ok := c.data[section]; ok {
		return v.(map[string]string), nil
	}
	return nil, errors.New("nonexist section " + section)
}

// SaveConfigFile save the config into file
func (c *JSONConfigContainer) SaveConfigFile(ctx context.Context, filename string) (err error) {
	// Write configuration file by filename.
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	return err
}

// Set writes a new value for key.
func (c *JSONConfigContainer) Set(ctx context.Context, key, val string) error {
	c.Lock()
	defer c.Unlock()
	c.data[key] = val
	return nil
}

// DIY returns the raw value by a given key.
func (c *JSONConfigContainer) DIY(ctx context.Context, key string) (v interface{}, err error) {
	val := c.getData(key)
	if val != nil {
		return val, nil
	}
	return nil, errors.New("not exist key")
}

// section.key or key
func (c *JSONConfigContainer) getData(key string) interface{} {
	if len(key) == 0 {
		return nil
	}

	c.RLock()
	defer c.RUnlock()

	sectionKeys := strings.Split(key, "::")
	if len(sectionKeys) >= 2 {
		curValue, ok := c.data[sectionKeys[0]]
		if !ok {
			return nil
		}
		for _, key := range sectionKeys[1:] {
			if v, ok := curValue.(map[string]interface{}); ok {
				if curValue, ok = v[key]; !ok {
					return nil
				}
			}
		}
		return curValue
	}
	if v, ok := c.data[key]; ok {
		return v
	}
	return nil
}

func init() {
	config.Register("json", &JSONConfig{})
}
