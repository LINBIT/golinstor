package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	logCfg     *LogCfg
	// log        *logrus.Entry

	Nodes               *NodeService
	ResourceDefinitions *ResourceDefinitionService
	Resources           *ResourceService
}

type LogCfg struct {
	Out       io.Writer
	Formatter logrus.Formatter
	Level     string
}

func NewClient(options ...func(*Client) error) (*Client, error) {
	httpClient := http.DefaultClient

	hostPort := "localhost:3370"
	controllers := os.Getenv("LS_CONTROLLERS")
	// we could ping them, for now use the first if possible
	if controllers != "" {
		hostPort = strings.Split(controllers, ",")[0]

		lsPrefix := "linstor://"
		if strings.HasPrefix(hostPort, lsPrefix) {
			hostPort = strings.TrimPrefix(hostPort, lsPrefix)
		}
	}

	if !strings.Contains(hostPort, ":") {
		hostPort += ":3370"
	}

	u := hostPort
	if !strings.HasPrefix(hostPort, "http://") {
		u = "http://" + hostPort
	}

	baseUrl, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	c := &Client{
		httpClient: httpClient,
		baseURL:    baseUrl,
	}
	l := &LogCfg{
		Level: logrus.WarnLevel.String(),
	}
	Log(l)(c)

	c.Nodes = &NodeService{client: c}
	c.ResourceDefinitions = &ResourceDefinitionService{client: c}
	c.Resources = &ResourceService{client: c}

	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// Options for the Client
// For example:
// u, _ := url.Parse("http://somehost:3370")
// c, _ := linstor.NewClient(linstor.BaseURL(u))

func BaseURL(URL *url.URL) func(*Client) error {
	return func(c *Client) error {
		c.baseURL = URL
		return nil
	}
}

func HTTPClient(httpClient *http.Client) func(*Client) error {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

func Log(logCfg *LogCfg) func(*Client) error {
	return func(c *Client) error {
		c.logCfg = logCfg
		level, err := logrus.ParseLevel(c.logCfg.Level)
		if err != nil {
			return err
		}
		logrus.SetLevel(level)
		if c.logCfg.Out == nil {
			c.logCfg.Out = os.Stderr
		}
		logrus.SetOutput(c.logCfg.Out)
		if c.logCfg.Formatter != nil {
			logrus.SetFormatter(c.logCfg.Formatter)
		}
		return nil
	}
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.baseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
		logrus.Debug(body)
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	// req.Header.Set("User-Agent", c.UserAgent)

	return req, nil
}

func (c *Client) do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		var rets []ApiCallRc
		if err = json.NewDecoder(resp.Body).Decode(&rets); err != nil {
			return nil, err
		}
		return nil, errors.New(rets[0].String())
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}
	return resp, err
}

// Higer Leve Abstractions

func (c *Client) doGET(ctx context.Context, url string, ret interface{}, opts ...*ListOpts) (*http.Response, error) {

	u, err := addOptions(url, genOptions(opts...))
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, req, ret)
}

func (c *Client) doPOST(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	return c.do(ctx, req, nil)
}

func (c *Client) doPUT(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	return c.do(ctx, req, nil)
}

func (c *Client) doDELETE(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("DELETE", url, body)
	if err != nil {
		return nil, err
	}

	return c.do(ctx, req, nil)
}

type ApiCallRc struct {
	// A masked error number
	RetCode        int64             `json:"ret_code"`
	Message        string            `json:"message"`
	Cause          string            `json:"cause,omitempty"`
	Details        string            `json:"details,omitempty"`
	Correction     string            `json:"correction,omitempty"`
	ErrorReportIds []string          `json:"error_report_ids,omitempty"`
	ObjRefs        map[string]string `json:"obj_refs,omitempty"`
}

func (rc *ApiCallRc) String() string {
	s := fmt.Sprintf("Message: '%s'", rc.Message)
	if rc.Cause != "" {
		s += fmt.Sprintf("; Cause: '%s'", rc.Cause)
	}
	if rc.Details != "" {
		s += fmt.Sprintf("; Details: '%s'", rc.Details)
	}
	if rc.Correction != "" {
		s += fmt.Sprintf("; Correction: '%s'", rc.Correction)
	}
	if len(rc.ErrorReportIds) > 0 {
		s += fmt.Sprintf("; Reports: '[%s]'", strings.Join(rc.ErrorReportIds, ","))
	}

	return s
}

type DeleteProps []string
type OverrideProps map[string]string
type PropsModify struct {
	DeleteProps   DeleteProps   `json:"delete_props,omitempty"`
	OverrideProps OverrideProps `json:"override_props,omitempty"`
}
