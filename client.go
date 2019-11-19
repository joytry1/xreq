package xreq

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// Config defines a config for Client.
type Config struct {
	Timeout   time.Duration
	Transport http.RoundTripper
}

// Client wraps a HTTP Client that support functional options
// and make HTTP requests easier.
// It also compatible with the http.Client.
type Client struct {
	hc     *http.Client
	config Config
	opt    []Option
}

var defaultClient = Client{
	hc: &http.Client{},
	config: Config{
		Timeout:   0,
		Transport: http.DefaultTransport,
	},
	opt: make([]Option, 0),
}

// NewClient return a Client instance.
func NewClient(conf Config, opt ...Option) *Client {
	return &Client{
		hc: &http.Client{
			Transport: conf.Transport,
			Timeout:   conf.Timeout,
		},
		config: conf,
		opt:    opt,
	}
}

// Get issues a GET with options to the specified URL
// and return *http.Response.
func Get(url string, opt ...Option) (*http.Response, error) {
	return defaultClient.Get(url, opt...)
}

// GetBytes issues GET with options to the specified URL
// and return the bytes of the resp.Body.
func GetBytes(url string, opt ...Option) (data []byte, code int, err error) {
	return defaultClient.GetBytes(url, opt...)
}

// Post issues a POST with options to the specified URL and return *http.Response.
// This method is compatible with the original usage
// please use Do or DoBytes as much as possible.
func Post(url, contentType string, body io.Reader, opt ...Option) (*http.Response, error) {
	return defaultClient.Post(url, contentType, body, opt...)
}

// PostBytes issues a POST with options to the specified URL
// and return the bytes of the resp.Body.
// This method is compatible with the original usage
// please use Do or DoBytes as much as possible.
func PostBytes(url, contentType string, body io.Reader, opt ...Option) (data []byte, code int, err error) {
	return defaultClient.PostBytes(url, contentType, body, opt...)
}

// Do method construct a HTTP request with options
// Example:
//
// resp, err := Do("http://localhost/api",
// 					WithPostJSON(v),
// 					WithCheckStatus(true))
// and return the *http.Response.
func Do(url string, opt ...Option) (*http.Response, error) {
	return defaultClient.Do(url, opt...)
}

// DoBytes method construct a HTTP request with options
// and return the bytes of resp.Body and http.StatusCode
func DoBytes(url string, opt ...Option) (data []byte, code int, err error) {
	return defaultClient.DoBytes(url, opt...)
}

// Get issues a GET with options to the specified URL
// and return *http.Response.
func (c *Client) Get(url string, opt ...Option) (*http.Response, error) {
	return c.Do(url, opt...)
}

// GetBytes issues GET with options to the specified URL
// and return the bytes of the resp.Body.
func (c *Client) GetBytes(url string, opt ...Option) (data []byte, code int, err error) {
	return c.DoBytes(url, opt...)
}

// Post issues a POST with options to the specified URL and return *http.Response.
// This method is compatible with the original usage
// please use Do or DoBytes as much as possible.
func (c *Client) Post(url, contentType string, body io.Reader, opt ...Option) (*http.Response, error) {
	ropt := make([]Option, len(opt)+2)
	ropt[0] = WithMethod(http.MethodPost)
	ropt[1] = WithBodyReader(contentType, body)
	copy(ropt[2:], opt)
	return c.Do(url, ropt...)
}

// PostBytes issues a POST with options to the specified URL
// and return the bytes of the resp.Body.
// This method is compatible with the original usage
// please use Do or DoBytes more likely.
func (c *Client) PostBytes(url, contentType string, body io.Reader, opt ...Option) (data []byte, code int, err error) {
	ropt := make([]Option, len(opt)+2)
	ropt[0] = WithMethod(http.MethodPost)
	ropt[1] = WithBodyReader(contentType, body)
	copy(ropt[2:], opt)
	return c.DoBytes(url, ropt...)
}

// Do method construct a HTTP request with options
// Example:
//
// resp, err := Do("http://localhost/api",
// 					WithPostJSON(v),
// 					WithCheckStatus(true))
// and return the *http.Response.
func (c *Client) Do(url string, opt ...Option) (*http.Response, error) {
	opts := &Options{}
	resp, err := c.do(opts, url, opt...)
	if err != nil {
		return nil, err
	}

	// NOTE method with return *http.Response does not check the resp.StatusCode
	// so it need to caller check the resp.StatusCode.
	// It maybe the caller would like to `if err != nil { return }`
	// and cause `resp.Body` doesn't closed.
	// if opts.checkStatus && resp.StatusCode != http.StatusOK {
	// 		err = fmt.Errorf("http status code: %d", resp.StatusCode)
	// }
	return resp, err
}

// DoBytes method construct a HTTP request with options
// and return the bytes of resp.Body and http.StatusCode.
func (c *Client) DoBytes(url string, opt ...Option) (data []byte, code int, err error) {
	opts := &Options{}
	resp, err := c.do(opts, url, opt...)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return data, resp.StatusCode, fmt.Errorf("read body error: %w", err)
	}

	// treat non-2xx as error will be better?
	if opts.checkStatus && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("http status code: %d", resp.StatusCode)
	}
	return data, resp.StatusCode, err
}

func (c *Client) do(opts *Options, url string, opt ...Option) (resp *http.Response, err error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request error: %w", err)
	}

	opts.Request = req
	opts.Values = req.URL.Query()
	opts.checkStatus = false

	allOpt := append(c.opt, opt...)
	for _, o := range allOpt {
		o(opts)
		if opts.Err != nil {
			return nil, fmt.Errorf("option exec error: %w", opts.Err)
		}
	}
	opts.Request.URL.RawQuery = opts.Values.Encode()

	return c.hc.Do(opts.Request)
}
