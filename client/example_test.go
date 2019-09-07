/*
* A REST client to interact with LINSTOR's REST API
* Copyright © 2019 LINBIT HA-Solutions GmbH
* Author: Roland Kammerer <roland.kammerer@linbit.com>
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package client_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/LINBIT/golinstor/client"
	"github.com/sirupsen/logrus"
)

func Example_simple() {
	ctx := context.TODO()

	u, err := url.Parse("http://controller:3370")
	if err != nil {
		log.Fatal(err)
	}

	logCfg := &client.LogCfg{
		Level: logrus.TraceLevel.String(),
	}

	c, err := client.NewClient(client.BaseURL(u), client.Log(logCfg))
	if err != nil {
		log.Fatal(err)
	}

	rs, err := c.Resources.GetAll(ctx, "foo")
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range rs {
		fmt.Printf("Resource with name '%s' on node with name '%s'\n", r.Name, r.NodeName)
	}
}

func Example_https() {
	ctx := context.TODO()

	u, err := url.Parse("https://controller:3371")
	if err != nil {
		log.Fatal(err)
	}

	// Be careful if that is really what you want!
	trSkipVerify := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	httpClient := &http.Client{
		Transport: trSkipVerify,
	}

	c, err := client.NewClient(client.BaseURL(u), client.HTTPClient(httpClient))
	if err != nil {
		log.Fatal(err)
	}

	rs, err := c.Resources.GetAll(ctx, "foo")
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range rs {
		fmt.Printf("Resource with name '%s' on node with name '%s'\n", r.Name, r.NodeName)
	}
}

func Example_httpsauth() {
	ctx := context.TODO()

	u, err := url.Parse("https://controller:3371")
	if err != nil {
		log.Fatal(err)
	}

	// Be careful if that is really what you want!
	trSkipVerify := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	httpClient := &http.Client{
		Transport: trSkipVerify,
	}

	c, err := client.NewClient(client.BaseURL(u), client.HTTPClient(httpClient),
		client.BasicAuth(&client.BasicAuthCfg{Username: "Username", Password: "Password"}))
	if err != nil {
		log.Fatal(err)
	}

	rs, err := c.Resources.GetAll(ctx, "foo")
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range rs {
		fmt.Printf("Resource with name '%s' on node with name '%s'\n", r.Name, r.NodeName)
	}
}
