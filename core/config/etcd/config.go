// Copyright 2020
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/mitchellh/mapstructure"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"

	"github.com/asish-tom/beego/v2/core/config"
	"github.com/asish-tom/beego/v2/core/logs"
)

type EtcdConfiger struct {
	prefix string
	client *clientv3.Client
	config.BaseConfiger
}

func newEtcdConfiger(client *clientv3.Client, prefix string) *EtcdConfiger {
	res := &EtcdConfiger{
		client: client,
		prefix: prefix,
	}

	res.BaseConfiger = config.NewBaseConfiger(res.reader)
	return res
}

// reader is a general implementation that read config from etcd.
func (e *EtcdConfiger) reader(ctx context.Context, key string) (string, error) {
	resp, err := get(e.client, e.prefix+key)
	if err != nil {
		return "", err
	}

	if resp.Count > 0 {
		return string(resp.Kvs[0].Value), nil
	}

	return "", nil
}

// Set do nothing and return an error
// I think write data to remote config center is not a good practice
func (e *EtcdConfiger) Set(key, val string) error {
	return errors.New("Unsupported operation")
}

// DIY return the original response from etcd
// be careful when you decide to use this
func (e *EtcdConfiger) DIY(key string) (interface{}, error) {
	return get(e.client, key)
}

// GetSection in this implementation, we use section as prefix
func (e *EtcdConfiger) GetSection(section string) (map[string]string, error) {
	var (
		resp *clientv3.GetResponse
		err  error
	)

	resp, err = e.client.Get(context.TODO(), e.prefix+section, clientv3.WithPrefix())

	if err != nil {
		return nil, fmt.Errorf("GetSection failed: %w", err)
	}
	res := make(map[string]string, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		res[string(kv.Key)] = string(kv.Value)
	}
	return res, nil
}

func (e *EtcdConfiger) SaveConfigFile(filename string) error {
	return errors.New("Unsupported operation")
}

// Unmarshaler is not very powerful because we lost the type information when we get configuration from etcd
// for example, when we got "5", we are not sure whether it's int 5, or it's string "5"
// TODO(support more complicated decoder)
func (e *EtcdConfiger) Unmarshaler(prefix string, obj interface{}, opt ...config.DecodeOption) error {
	res, err := e.GetSection(prefix)
	if err != nil {
		return fmt.Errorf("could not read config with prefix: %s: %w", prefix, err)
	}

	prefixLen := len(e.prefix + prefix)
	m := make(map[string]string, len(res))
	for k, v := range res {
		m[k[prefixLen:]] = v
	}
	return mapstructure.Decode(m, obj)
}

// Sub return an sub configer.
func (e *EtcdConfiger) Sub(key string) (config.Configer, error) {
	return newEtcdConfiger(e.client, e.prefix+key), nil
}

// TODO remove this before release v2.0.0
func (e *EtcdConfiger) OnChange(key string, fn func(value string)) {
	buildOptsFunc := func() []clientv3.OpOption {
		return []clientv3.OpOption{}
	}

	rch := e.client.Watch(context.Background(), e.prefix+key, buildOptsFunc()...)
	go func() {
		for {
			for resp := range rch {
				if err := resp.Err(); err != nil {
					logs.Error("listen to key but got error callback", err)
					break
				}

				for _, e := range resp.Events {
					if e.Kv == nil {
						continue
					}
					fn(string(e.Kv.Value))
				}
			}
			time.Sleep(time.Second)
			rch = e.client.Watch(context.Background(), e.prefix+key, buildOptsFunc()...)
		}
	}()
}

type EtcdConfigerProvider struct{}

// Parse = ParseData([]byte(key))
// key must be json
func (provider *EtcdConfigerProvider) Parse(key string) (config.Configer, error) {
	return provider.ParseData([]byte(key))
}

// ParseData try to parse key as clientv3.Config, using this to build etcdClient
func (provider *EtcdConfigerProvider) ParseData(data []byte) (config.Configer, error) {
	cfg := &clientv3.Config{}
	err := json.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("parse data to etcd config failed, please check your input: %w", err)
	}

	cfg.DialOptions = []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	client, err := clientv3.New(*cfg)
	if err != nil {
		return nil, fmt.Errorf("create etcd client failed: %w", err)
	}

	return newEtcdConfiger(client, ""), nil
}

func get(client *clientv3.Client, key string) (*clientv3.GetResponse, error) {
	var (
		resp *clientv3.GetResponse
		err  error
	)
	resp, err = client.Get(context.Background(), key)

	if err != nil {
		return nil, fmt.Errorf("read config from etcd with key %s failed: %w", key, err)
	}
	return resp, err
}

func init() {
	config.Register("etcd", &EtcdConfigerProvider{})
}
