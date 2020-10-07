package client_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"

	"github.com/LINBIT/golinstor/client"
	log "github.com/sirupsen/logrus"
)

func Example_simple() {
	ctx := context.TODO()

	u, err := url.Parse("http://controller:3370")
	if err != nil {
		log.Fatal(err)
	}

	c, err := client.NewClient(client.BaseURL(u), client.Log(log.StandardLogger()))
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

func Example_error() {
	ctx := context.TODO()

	u, err := url.Parse("http://controller:3370")
	if err != nil {
		log.Fatal(err)
	}

	c, err := client.NewClient(client.BaseURL(u), client.Log(log.StandardLogger()))
	if err != nil {
		log.Fatal(err)
	}

	rs, err := c.Resources.GetAll(ctx, "foo")
	if errs, ok := err.(client.ApiCallError); ok {
		log.Error("A LINSTOR API error occurred:")
		for i, e := range errs {
			log.Errorf("  Message #%d:", i)
			log.Errorf("    Code: %d", e.RetCode)
			log.Errorf("    Message: %s", e.Message)
			log.Errorf("    Cause: %s", e.Cause)
			log.Errorf("    Details: %s", e.Details)
			log.Errorf("    Correction: %s", e.Correction)
			log.Errorf("    Error Reports: %v", e.ErrorReportIds)
		}
		return
	}
	if err != nil {
		log.Fatalf("Some other error occurred: %s", err.Error())
	}

	for _, r := range rs {
		fmt.Printf("Resource with name '%s' on node with name '%s'\n", r.Name, r.NodeName)
	}
}

func Example_events() {
	ctx := context.TODO()

	u, err := url.Parse("http://controller:3370")
	if err != nil {
		log.Fatal(err)
	}

	c, err := client.NewClient(client.BaseURL(u))
	if err != nil {
		log.Fatal(err)
	}

	mayPromoteStream, err := c.Events.DRBDPromotion(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer mayPromoteStream.Close()

	for ev := range mayPromoteStream.Events {
		fmt.Printf("Resource '%s' on node with name '%s' may promote: %t\n", ev.ResourceName, ev.NodeName, ev.MayPromote)
	}
}
