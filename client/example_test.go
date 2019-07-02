package client_test

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/LINBIT/golinstor/client"
	"github.com/sirupsen/logrus"
)

func Example() {
	ctx := context.Background()

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
