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

func ExampleSimple() {
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

func ExampleHTTPS() {
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

func ExampleHTTPSLDAP() {
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
