package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2/google"
)

type QueryResponse struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
	Error  string `json:"error"`
}

// JSON response is decoded two times to create Date struct.
// 1st decode is for populating the ResultRaw field.
// 2nd decode is for populating the Result field depending on the result type.
// Format: https://prometheus.io/docs/prometheus/latest/querying/api/#expression-query-result-formats
type Data struct {
	ResultType string          `json:"resultType"`
	ResultRaw  json.RawMessage `json:"result"`
	// Result could contain either ResultScalar, ResultString, ResultVector, or ResultMatrix.
	Result any `json:"-"`
}

type ResultScalar []any
type ResultString []any
type ResultVector []VectorTimeSeries
type ResultMatrix []MatrixTimeSeries

type VectorTimeSeries struct {
	Metric map[string]string `json:"metric"`
	Point  []any             `json:"value"`
}

type MatrixTimeSeries struct {
	Metric map[string]string `json:"metric"`
	Points [][]any           `json:"values"`
}

type Client struct {
	baseURL string
	header  http.Header
	client  *http.Client
}

func NewClient(ctx context.Context, baseURL string, projectID string, headers string) (*Client, error) {
	httpClient := http.DefaultClient

	// For Google Cloud Monitoring
	if projectID != "" {
		baseURL = fmt.Sprintf("https://monitoring.googleapis.com/v1/projects/%s/location/global/prometheus", projectID)
		googleClient, err := google.DefaultClient(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, err
		}
		httpClient = googleClient
	}

	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %v", err)
	}

	var header http.Header
	if headers != "" {
		var err error
		header, err = parseHeaderString(headers)
		if err != nil {
			return nil, err
		}
	}

	return &Client{
		baseURL: baseURL,
		header:  header,
		client:  httpClient,
	}, nil
}

func (c *Client) Query(q string) (*QueryResponse, error) {
	u, _ := url.Parse(c.baseURL) // ignore error since baseURL is already validated
	u = u.JoinPath("/api/v1/query")
	queryParams := url.Values{}
	queryParams.Add("query", q)
	u.RawQuery = queryParams.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header = c.header

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var qr QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return nil, err
	}

	if qr.Status == "error" {
		return nil, errors.New(qr.Error)
	}

	switch qr.Data.ResultType {
	case "scalar":
		var result ResultScalar
		if err := json.Unmarshal(qr.Data.ResultRaw, &result); err != nil {
			return nil, err
		}
		qr.Data.Result = result
	case "string":
		var result ResultString
		if err := json.Unmarshal(qr.Data.ResultRaw, &result); err != nil {
			return nil, err
		}
		qr.Data.Result = result
	case "vector":
		var result ResultVector
		if err := json.Unmarshal(qr.Data.ResultRaw, &result); err != nil {
			return nil, err
		}
		qr.Data.Result = result
	case "matrix":
		var result ResultMatrix
		if err := json.Unmarshal(qr.Data.ResultRaw, &result); err != nil {
			return nil, err
		}
		qr.Data.Result = result
	default:
		return nil, fmt.Errorf("unsupported result type: %q", qr.Data.ResultType)
	}

	return &qr, nil
}

func parseHeaderString(headers string) (http.Header, error) {
	header := make(http.Header, 0)
	for _, h := range strings.Split(headers, ",") {
		key, val, found := strings.Cut(h, ":")
		if !found {
			return header, fmt.Errorf("invalid header: %q", h)
		}
		header.Add(strings.TrimSpace(key), strings.TrimSpace(val))
	}
	return header, nil
}
