package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/olekukonko/tablewriter"
)

const (
	exitCodeSuccess = 0
	exitCodeError   = 1

	defaultPrompt = "promql> "
)

type CLI struct {
	client *Client
	in     io.ReadCloser
	out    io.Writer
}

func NewCLI(url, project, headers string, in io.ReadCloser, out io.Writer) (*CLI, error) {
	ctx := context.Background()
	client, err := NewClient(ctx, url, project, headers)
	if err != nil {
		return nil, err
	}

	return &CLI{
		client: client,
		in:     in,
		out:    out,
	}, nil
}

func (c *CLI) RunInteractive() int {
	rl, err := readline.NewEx(&readline.Config{
		Stdin:       c.in,
		HistoryFile: "/tmp/promql_cli_history",
	})
	if err != nil {
		return c.ExitOnError(err)
	}
	rl.SetPrompt(defaultPrompt)

	for {
		input, err := c.ReadInput(rl)
		if err == io.EOF {
			return c.Exit()
		}
		if err != nil {
			return c.ExitOnError(err)
		}

		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			return c.Exit()
		}

		stop := c.PrintProgressingMark()
		resp, err := c.client.Query(input)
		stop()
		if err != nil {
			c.PrintInteractiveError(err)
			continue
		}

		table := buildTable(resp)
		if len(table.Rows) > 0 {
			w := tablewriter.NewWriter(c.out)
			w.SetAutoFormatHeaders(false)
			w.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
			w.SetAlignment(tablewriter.ALIGN_LEFT)
			w.SetAutoWrapText(false)
			for _, row := range table.Rows {
				w.Append(row.Columns)
			}
			w.SetHeader(table.Header)
			w.Render()
			fmt.Fprintf(c.out, "%d values in result\n\n", len(table.Rows))
		} else {
			fmt.Fprintf(c.out, "Empty result\n\n")
		}
	}
}

func (c *CLI) ReadInput(rl *readline.Instance) (string, error) {
	defer rl.SetPrompt(defaultPrompt)

	for {
		line, err := rl.Readline()
		if err != nil {
			return "", err
		}
		if line == "" {
			continue
		}

		return strings.TrimSpace(line), nil
	}
}

func (c *CLI) Exit() int {
	fmt.Fprintln(c.out, "Bye")
	return exitCodeSuccess
}

func (c *CLI) ExitOnError(err error) int {
	fmt.Fprintf(c.out, "ERROR: %s\n", err)
	return exitCodeError
}

func (c *CLI) PrintInteractiveError(err error) {
	fmt.Fprintf(c.out, "ERROR: %s\n", err)
}

func (c *CLI) PrintProgressingMark() func() {
	progressMarks := []string{`-`, `\`, `|`, `/`}
	ticker := time.NewTicker(time.Millisecond * 100)
	go func() {
		i := 0
		for {
			<-ticker.C
			mark := progressMarks[i%len(progressMarks)]
			fmt.Fprintf(c.out, "\r%s", mark)
			i++
		}
	}()

	stop := func() {
		ticker.Stop()
		fmt.Fprintf(c.out, "\r") // clear progressing mark
	}
	return stop
}

type Table struct {
	Header []string
	Rows   []Row
}

type Row struct {
	Columns []string
}

func buildTable(qr *QueryResponse) *Table {
	table := Table{}

	if len(qr.Data.ResultRaw) == 0 {
		return &table
	}

	switch result := qr.Data.Result.(type) {
	case ResultScalar:
		// Add header columns.
		table.Header = []string{"timestamp", "value"}

		// Add row.
		timestamp := result[0].(float64)
		value := result[1].(string)
		table.Rows = []Row{{Columns: []string{formatTimestamp(timestamp), value}}}
		return &table
	case ResultString:
		// Add header columns.
		table.Header = []string{"timestamp", "value"}

		// Add row.
		timestamp := result[0].(float64)
		value := result[1].(string)
		table.Rows = []Row{{Columns: []string{formatTimestamp(timestamp), value}}}
		return &table
	case ResultVector:
		if len(result) == 0 {
			return &table
		}

		// Add header columns.
		table.Header = []string{"timestamp"}
		table.Header = append(table.Header, sortedLabelNames(result[0].Metric)...)
		table.Header = append(table.Header, "value")

		// Add rows.
		for _, timeseries := range result {
			var row Row
			timestamp := timeseries.Point[0].(float64)
			value := timeseries.Point[1].(string)

			row.Columns = append(row.Columns, formatTimestamp(timestamp))
			for _, labelName := range sortedLabelNames(timeseries.Metric) {
				row.Columns = append(row.Columns, timeseries.Metric[labelName])
			}
			row.Columns = append(row.Columns, value)
			table.Rows = append(table.Rows, row)
		}
		return &table
	case ResultMatrix:
		if len(result) == 0 {
			return &table
		}

		// Add header columns.
		table.Header = []string{"timestamp"}
		table.Header = append(table.Header, sortedLabelNames(result[0].Metric)...)
		table.Header = append(table.Header, "value")

		// Add rows.
		for _, timeseries := range result {
			for _, point := range timeseries.Points {
				timestamp := point[0].(float64)
				value := point[1].(string)

				var row Row
				row.Columns = append(row.Columns, formatTimestamp(timestamp))
				for _, labelName := range sortedLabelNames(timeseries.Metric) {
					row.Columns = append(row.Columns, timeseries.Metric[labelName])
				}
				row.Columns = append(row.Columns, value)
				table.Rows = append(table.Rows, row)
			}
		}
		return &table
	default:
		// Unreachable.
		return &table
	}
}

func sortedLabelNames(labels map[string]string) []string {
	var labelNames []string
	for l := range labels {
		labelNames = append(labelNames, l)
	}
	sort.Slice(labelNames, func(i, j int) bool {
		labelI := labelNames[i]
		labelJ := labelNames[j]

		// metric name should be at leftmost
		if labelI == "__name__" {
			return true
		}
		if labelJ == "__name__" {
			return false
		}

		// "le", which is used for histogram metrics, should be at rightmost
		if labelI == "le" {
			return false
		}
		if labelJ == "le" {
			return true
		}

		return sort.StringsAreSorted([]string{labelI, labelJ})
	})
	return labelNames
}

func formatTimestamp(timestamp float64) string {
	t := time.UnixMicro(int64(timestamp * 1_000_000))
	return t.Format(time.RFC3339Nano)
}
