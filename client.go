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
	Data   Data
}

type Data struct {
	ResultType string       `json:"resultType"`
	Result     []TimeSeries `json:"result"`
}

type TimeSeries struct {
	Metric map[string]string `json:"metric"`
	Point  []any             `json:"value"`
	Points [][]any           `json:"values"`
}

type Client struct {
	baseURL string
	// client *http.Client
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

	return &qr, nil
}
