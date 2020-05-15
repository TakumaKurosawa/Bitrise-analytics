package cmd

import (
	"bytes"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type apiClient struct {
	httpClient *http.Client
	Logger     *log.Logger
}

type apiParams struct {
	method      string
	url         *url.URL
	path        string
	query       url.Values
	header      string
	contentType string
	body        string
}

func newAPIClient() *apiClient {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
		},
		Timeout: 10 * time.Second,
	}

	return &apiClient{
		httpClient: client,
	}
}

func (ac *apiClient) doRequest(params *apiParams) (*http.Response, string, error) {
	req, _ := http.NewRequest(params.method, params.url.String(), strings.NewReader(params.body))
	req.Header.Set("Authorization", params.header)

	res, _ := ac.httpClient.Do(req)
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)

	return res, bytes.NewBuffer(b).String(), nil
}
