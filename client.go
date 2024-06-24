package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type QueryResponse struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

// JSON response is decoded two times to create Date struct.
// 1st decode is for populating the Result field.
// 2nd decode is for populating the ResultScalar/ResultVector/ResultMatrix fields depending on the result type.
// Format: https://prometheus.io/docs/prometheus/latest/querying/api/#expression-query-result-formats
type Data struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`

	ResultScalar []any
	ResultVector []VectorTimeSeries
	ResultMatrix []MatrixTimeSeries
}

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
}

func NewClient(ctx context.Context, baseURL string) (*Client, error) {
	return &Client{
		baseURL: baseURL,
	}, nil
}

func (c *Client) Query(q string) (*QueryResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %v", err)
	}
	u.Path = "/api/v1/query"
	queryParams := url.Values{}
	queryParams.Add("query", q)
	u.RawQuery = queryParams.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var qr QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return nil, err
	}

	switch qr.Data.ResultType {
	case "scalar":
		var resultScalar []any
		if err := json.Unmarshal(qr.Data.Result, &resultScalar); err != nil {
			return nil, err
		}
		qr.Data.ResultScalar = resultScalar
	case "vector":
		var resultVector []VectorTimeSeries
		if err := json.Unmarshal(qr.Data.Result, &resultVector); err != nil {
			return nil, err
		}
		qr.Data.ResultVector = resultVector
	case "matrix":
		var resultMatrix []MatrixTimeSeries
		if err := json.Unmarshal(qr.Data.Result, &resultMatrix); err != nil {
			return nil, err
		}
		qr.Data.ResultMatrix = resultMatrix
	default:
		return nil, fmt.Errorf("unsupported result type: %q", qr.Data.ResultType)
	}

	return &qr, nil
}
