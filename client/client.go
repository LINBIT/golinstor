// A REST client to interact with LINSTOR's REST API
// Copyright (C) LINBIT HA-Solutions GmbH
// All Rights Reserved.
// Author: Roland Kammerer <roland.kammerer@linbit.com>
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/donovanhide/eventsource"
	"golang.org/x/time/rate"
	"moul.io/http2curl/v2"

	linstor "github.com/LINBIT/golinstor"
)

// Client is a struct representing a LINSTOR REST client.
type Client struct {
	httpClient    *http.Client
	basicAuth     *BasicAuthCfg
	bearerToken   string
	userAgent     string
	controllersMu sync.Mutex
	controllers   []*url.URL
	lim           *rate.Limiter
	log           interface{} // must be either Logger or LeveledLogger

	Nodes                  NodeProvider
	ResourceDefinitions    ResourceDefinitionProvider
	Resources              ResourceProvider
	ResourceGroups         ResourceGroupProvider
	StoragePoolDefinitions StoragePoolDefinitionProvider
	Encryption             EncryptionProvider
	Controller             ControllerProvider
	Events                 EventProvider
	Vendor                 VendorProvider
	Remote                 RemoteProvider
	Backup                 BackupProvider
	KeyValueStore          KeyValueStoreProvider
	Connections            ConnectionProvider
}

// Logger represents a standard logger interface
type Logger interface {
	Printf(string, ...interface{})
}

// LeveledLogger interface implements the basic methods that a logger library needs
type LeveledLogger interface {
	Errorf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Warnf(string, ...interface{})
}

type BasicAuthCfg struct {
	Username, Password string
}

// const errors as in https://dave.cheney.net/2016/04/07/constant-errors
type clientError string

func (e clientError) Error() string { return string(e) }

const (
	// NotFoundError is the error type returned in case of a 404 error. This is required to test for this kind of error.
	NotFoundError = clientError("404 Not Found")
	// Name of the environment variable that stores the certificate used for TLS client authentication
	UserCertEnv = "LS_USER_CERTIFICATE"
	// Name of the environment variable that stores the key used for TLS client authentication
	UserKeyEnv = "LS_USER_KEY"
	// Name of the environment variable that stores the certificate authority for the LINSTOR HTTPS API
	RootCAEnv = "LS_ROOT_CA"
	// Name of the environment variable that holds the URL(s) of LINSTOR controllers
	ControllerUrlEnv = "LS_CONTROLLERS"
	// Name of the environment variable that holds the username for authentication
	UsernameEnv = "LS_USERNAME"
	// Name of the environment variable that holds the password for authentication
	PasswordEnv = "LS_PASSWORD"
	// Name of the environment variable that points to the file containing the token for authentication
	BearerTokenFileEnv = "LS_BEARER_TOKEN_FILE"
)

// For example:
// u, _ := url.Parse("http://somehost:3370")
// c, _ := linstor.NewClient(linstor.BaseURL(u))

// Option configures a LINSTOR Client
type Option func(*Client) error

// BaseURL is a client's option to set the baseURL of the REST client.
//
// If multiple URLs are provided, each is tried in turn.
func BaseURL(urls ...*url.URL) Option {
	return func(c *Client) error {
		c.controllers = urls
		return nil
	}
}

// BasicAuth is a client's option to set username and password for the REST client.
func BasicAuth(basicauth *BasicAuthCfg) Option {
	return func(c *Client) error {
		c.basicAuth = basicauth
		return nil
	}
}

// HTTPClient is a client's option to set a specific http.Client.
func HTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

// Log is a client's option to set a Logger
func Log(logger interface{}) Option {
	return func(c *Client) error {
		switch logger.(type) {
		case Logger, LeveledLogger, nil:
			c.log = logger
		default:
			return errors.New("Invalid logger type, expected Logger or LeveledLogger")
		}
		return nil
	}
}

// Limiter to use when making queries.
// Mutually exclusive with Limit, last applied option wins.
func Limiter(limiter *rate.Limiter) Option {
	return func(c *Client) error {
		if limiter.Burst() == 0 && limiter.Limit() != rate.Inf {
			return fmt.Errorf("invalid rate limit, burst must not be zero for non-unlimited rates")
		}
		c.lim = limiter
		return nil
	}
}

// Limit is the client's option to set number of requests per second and
// max number of bursts.
// Mutually exclusive with Limiter, last applied option wins.
// Deprecated: Use Limiter instead.
func Limit(r rate.Limit, b int) Option {
	return Limiter(rate.NewLimiter(r, b))
}

func Controllers(controllers []string) Option {
	return func(c *Client) error {
		var err error
		c.controllers, err = parseURLs(controllers)
		return err
	}
}

// BearerToken configures authentication via the given token send in the Authorization Header.
// If set, this will override any authentication happening via Basic Authentication.
func BearerToken(token string) Option {
	return func(c *Client) error {
		c.bearerToken = token
		return nil
	}
}

// UserAgent sets the User-Agent header for every request to the given string.
func UserAgent(ua string) Option {
	return func(c *Client) error {
		c.userAgent = ua
		return nil
	}
}

// buildHttpClient constructs an HTTP client which will be used to connect to
// the LINSTOR controller. It recongnizes some environment variables which can
// be used to configure the HTTP client at runtime. If an invalid key or
// certificate is passed, an error is returned.
// If none or not all of the environment variables are passed, the default
// client is used as a fallback.
func buildHttpClient() (*http.Client, error) {
	certPEM, cert := os.LookupEnv(UserCertEnv)
	keyPEM, key := os.LookupEnv(UserKeyEnv)
	caPEM, ca := os.LookupEnv(RootCAEnv)

	if key != cert {
		return nil, fmt.Errorf("'%s', '%s': specify both or none", UserKeyEnv, UserCertEnv)
	}

	if !cert && !key && !ca {
		// Non of the special variables was set -> if TLS is used, default configuration can be used
		return http.DefaultClient, nil
	}

	tlsConfig := &tls.Config{}

	if ca {
		caPool := x509.NewCertPool()
		ok := caPool.AppendCertsFromPEM([]byte(caPEM))
		if !ok {
			return nil, fmt.Errorf("failed to get a valid certificate from '%s'", RootCAEnv)
		}
		tlsConfig.RootCAs = caPool
	}

	if key && cert {
		keyPair, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to load keys: %w", err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, keyPair)
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}, nil
}

// Return the default scheme to access linstor
// If one of the HTTPS environment variables is set, will return "https".
// If not, will return "http"
func defaultScheme() string {
	_, ca := os.LookupEnv(RootCAEnv)
	_, cert := os.LookupEnv(UserCertEnv)
	_, key := os.LookupEnv(UserKeyEnv)
	if ca || cert || key {
		return "https"
	}
	return "http"
}

const defaultHost = "localhost"

// Return the default port to access linstor.
// Defaults are:
// "https": 3371
// "http":  3370
func defaultPort(scheme string) string {
	if scheme == "https" {
		return "3371"
	}
	return "3370"
}

func parseBaseURL(urlString string) (*url.URL, error) {
	// Check scheme
	urlSplit := strings.Split(urlString, "://")

	if len(urlSplit) == 1 {
		if urlSplit[0] == "" {
			urlSplit[0] = defaultHost
		}
		urlSplit = []string{defaultScheme(), urlSplit[0]}
	}

	if len(urlSplit) != 2 {
		return nil, fmt.Errorf("URL with multiple scheme separators. parts: %v", urlSplit)
	}
	scheme, endpoint := urlSplit[0], urlSplit[1]
	switch scheme {
	case "linstor":
		scheme = defaultScheme()
	case "linstor+ssl":
		scheme = "https"
	}

	// Check port
	endpointSplit := strings.Split(endpoint, ":")
	if len(endpointSplit) == 1 {
		endpointSplit = []string{endpointSplit[0], defaultPort(scheme)}
	}
	if len(endpointSplit) != 2 {
		return nil, fmt.Errorf("URL with multiple port separators. parts: %v", endpointSplit)
	}
	host, port := endpointSplit[0], endpointSplit[1]

	return url.Parse(fmt.Sprintf("%s://%s:%s", scheme, host, port))
}

func parseURLs(urls []string) ([]*url.URL, error) {
	var result []*url.URL
	for _, controller := range urls {
		url, err := parseBaseURL(controller)
		if err != nil {
			return nil, err
		}
		result = append(result, url)
	}

	return result, nil
}

// NewClient takes an arbitrary number of options and returns a Client or an error.
// It recognizes several environment variables which can be used to configure
// the client at runtime:
//
// - LS_CONTROLLERS: a comma-separated list of LINSTOR controllers to connect to.
//
// - LS_USERNAME, LS_PASSWORD: can be used to authenticate against the LINSTOR
// controller using HTTP basic authentication.
//
// - LS_USER_CERTIFICATE, LS_USER_KEY, LS_ROOT_CA: can be used to enable TLS on
// the HTTP client, enabling encrypted communication with the LINSTOR controller.
//
// - LS_BEARER_TOKEN_FILE: can be set to a file containing the bearer token used
// for authentication.
//
// Options passed to NewClient take precedence over options passed in via
// environment variables.
func NewClient(options ...Option) (*Client, error) {
	httpClient, err := buildHttpClient()
	if err != nil {
		return nil, fmt.Errorf("failed to build http client: %w", err)
	}

	c := &Client{
		httpClient: httpClient,
		basicAuth: &BasicAuthCfg{
			Username: os.Getenv(UsernameEnv),
			Password: os.Getenv(PasswordEnv),
		},
		lim: rate.NewLimiter(rate.Inf, 0),
		log: log.New(os.Stderr, "", 0),
	}

	c.Nodes = &NodeService{client: c}
	c.ResourceDefinitions = &ResourceDefinitionService{client: c}
	c.Resources = &ResourceService{client: c}
	c.Encryption = &EncryptionService{client: c}
	c.ResourceGroups = &ResourceGroupService{client: c}
	c.StoragePoolDefinitions = &StoragePoolDefinitionService{client: c}
	c.Controller = &ControllerService{client: c}
	c.Events = &EventService{client: c}
	c.Vendor = &VendorService{client: c}
	c.Remote = &RemoteService{client: c}
	c.Backup = &BackupService{client: c}
	c.KeyValueStore = &KeyValueStoreService{client: c}
	c.Connections = &ConnectionService{client: c}

	if path, ok := os.LookupEnv(BearerTokenFileEnv); ok {
		token, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read token from file: %w", err)
		}

		c.bearerToken = string(token)
	}

	for _, opt := range options {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if len(c.controllers) == 0 {
		// if not already set by option, get from environment...
		controllersStr := os.Getenv(ControllerUrlEnv)
		if controllersStr == "" {
			// ... or fall back to defaults
			controllersStr = fmt.Sprintf("%v://%v:%v", defaultScheme(), defaultHost, defaultPort(defaultScheme()))
		}

		c.controllers, err = parseURLs(strings.Split(controllersStr, ","))
		if err != nil {
			return nil, fmt.Errorf("failed to parse controller URLs: %w", err)
		}
	}

	return c, nil
}

// BaseURL returns the current controllers URL.
func (c *Client) BaseURL() *url.URL {
	c.controllersMu.Lock()
	defer c.controllersMu.Unlock()
	return c.controllers[0]
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL().ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		// Use json.Marshal instead of encoding to the buffer directly; json.Encoder.Encode() adds a newline
		// at the end, which is annoying for logging.
		jsonBuf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
		}
		buf = bytes.NewBuffer(jsonBuf)
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	username := c.basicAuth.Username
	if username != "" {
		req.SetBasicAuth(username, c.basicAuth.Password)
	}

	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	return req, nil
}

func (c *Client) curlify(req *http.Request) (string, error) {
	cc, err := http2curl.GetCurlCommand(req)
	if err != nil {
		return "", err
	}
	return cc.String(), nil
}

// findRespondingController scans the list of controllers for a working LINSTOR controller.
//
// After this returns successfully, the first controllers entry will point to the working controller.
//
// If no controller could be reached, an error combining all attempts is returned.
func (c *Client) findRespondingController() error {
	if len(c.controllers) <= 1 {
		return nil
	}

	c.controllersMu.Lock()
	defer c.controllersMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	errs := make([]error, len(c.controllers))
	var wg sync.WaitGroup
	wg.Add(len(c.controllers))
	for i := range c.controllers {
		i := i
		go func() {
			defer wg.Done()
			conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", c.controllers[i].Host)
			if err != nil {
				errs[i] = fmt.Errorf("failed to dial '%s': %w", c.controllers[i].Host, err)
				return
			}
			_ = conn.Close()
			// Success -> we can cancel the other goroutines
			cancel()
		}()
	}

	wg.Wait()
	cancel() // just to make linters happy

	for i := range errs {
		if errs[i] == nil {
			tmp := c.controllers[i]
			c.controllers[0] = c.controllers[i]
			c.controllers[i] = tmp
			return nil
		}
	}

	return fmt.Errorf("could not connect to any controller: %w", errors.Join(errs...))
}

func (c *Client) logCurlify(req *http.Request) {
	var msg string
	if curl, err := c.curlify(req); err != nil {
		msg = err.Error()
	} else {
		msg = curl
	}

	switch l := c.log.(type) {
	case LeveledLogger:
		l.Debugf("%s", msg)
	case Logger:
		l.Printf("[DEBUG] %s", msg)
	}
}

func (c *Client) retry(origErr error, req *http.Request) (*http.Response, error) {
	// only retry on network errors and if we even have another controller to choose from
	var netError net.Error
	if !errors.As(origErr, &netError) || len(c.controllers) <= 1 {
		return nil, origErr
	}

	e := c.findRespondingController()
	// if findRespondingController failed, or we just got the same base URL, don't bother retrying
	if e != nil {
		return nil, origErr
	}

	req.URL.Host = c.BaseURL().Host
	req.URL.Scheme = c.BaseURL().Scheme
	return c.httpClient.Do(req)
}

// do sends a prepared http.Request and returns the http.Response. If an HTTP error occurs, the parsed error is
// returned. Otherwise, the response is returned as-is. The caller is responsible for closing the response body in
// the non-error case.
func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.lim.Wait(ctx); err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	c.logCurlify(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// if this was a connectivity issue, attempt a retry
		resp, err = c.retry(err, req)
		if err != nil {
			return nil, err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		// If we get an error, we handle the body ourselves, so we also have to close it.
		defer resp.Body.Close()
		msg := fmt.Sprintf("Status code not within 200 to 400, but %d (%s)\n",
			resp.StatusCode, http.StatusText(resp.StatusCode))
		switch l := c.log.(type) {
		case LeveledLogger:
			l.Debugf("%s", msg)
		case Logger:
			l.Printf("[DEBUG] %s", msg)
		}
		if resp.StatusCode == 404 {
			return nil, NotFoundError
		}

		var rets ApiCallError
		if err = json.NewDecoder(resp.Body).Decode(&rets); err != nil {
			return nil, err
		}
		return nil, rets
	}
	return resp, err
}

// doJSON sends a prepared http.Request and returns the http.Response. If out is provided, the response body is
// JSON-decoded into out.
func (c *Client) doJSON(ctx context.Context, req *http.Request, out any) (*http.Response, error) {
	req.Header.Set("Accept", "application/json")
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if out != nil {
		err = json.NewDecoder(resp.Body).Decode(out)
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
	return c.doJSON(ctx, req, ret)
}

func (c *Client) doEvent(ctx context.Context, url, lastEventId string) (*eventsource.Stream, error) {
	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req = req.WithContext(ctx)

	stream, err := eventsource.SubscribeWith(lastEventId, c.httpClient, req)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (c *Client) doPOST(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	return c.doJSON(ctx, req, nil)
}

func (c *Client) doPUT(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	return c.doJSON(ctx, req, nil)
}

func (c *Client) doPATCH(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("PATCH", url, body)
	if err != nil {
		return nil, err
	}

	return c.doJSON(ctx, req, nil)
}

func (c *Client) doDELETE(ctx context.Context, url string, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("DELETE", url, body)
	if err != nil {
		return nil, err
	}

	return c.doJSON(ctx, req, nil)
}

func (c *Client) doOPTIONS(ctx context.Context, url string, ret interface{}, body interface{}) (*http.Response, error) {
	req, err := c.newRequest("OPTIONS", url, body)
	if err != nil {
		return nil, err
	}

	return c.doJSON(ctx, req, ret)
}

// ApiCallRc represents the struct returned by LINSTOR, when accessing its REST API.
type ApiCallRc struct {
	// A masked error number
	RetCode int64  `json:"ret_code"`
	Message string `json:"message"`
	// Cause of the error
	Cause string `json:"cause,omitempty"`
	// Details to the error message
	Details string `json:"details,omitempty"`
	// Possible correction options
	Correction string `json:"correction,omitempty"`
	// List of error report ids related to this api call return code.
	ErrorReportIds []string `json:"error_report_ids,omitempty"`
	// Map of objection that have been involved by the operation.
	ObjRefs map[string]string `json:"obj_refs,omitempty"`
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

// Is can be used to check the return code against a given mask. Since LINSTOR
// return codes are designed to be machine readable, this can be used to check
// for a very specific type of error.
// Refer to package apiconsts.go in package linstor for a list of possible
// mask values.
func (rc *ApiCallRc) Is(mask uint64) bool {
	return (uint64(rc.RetCode) & (linstor.MaskError | linstor.MaskBitsCode)) == mask
}

// DeleteProps is a slice of properties to delete.
type DeleteProps []string

// OverrideProps is a map of properties to modify (key/value pairs)
type OverrideProps map[string]string

// Namespaces to delete
type DeleteNamespaces []string

// GenericPropsModify is a struct combining DeleteProps and OverrideProps
type GenericPropsModify struct {
	DeleteProps      DeleteProps      `json:"delete_props,omitempty"`
	OverrideProps    OverrideProps    `json:"override_props,omitempty"`
	DeleteNamespaces DeleteNamespaces `json:"delete_namespaces,omitempty"`
}
