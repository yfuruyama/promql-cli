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

type Cli struct {
	url string
	in  io.ReadCloser
	out io.Writer
}

func NewCli(url string, in io.ReadCloser, out io.Writer) (*Cli, error) {
	return &Cli{
		url: url,
		in:  in,
		out: out,
	}, nil
}

func (c *Cli) RunInteractive() int {
	rl, err := readline.NewEx(&readline.Config{
		Stdin:       c.in,
		HistoryFile: "/tmp/promql_cli_history",
	})
	if err != nil {
		return c.ExitOnError(err)
	}
	rl.SetPrompt(defaultPrompt)

	ctx := context.Background()
	client, err := NewClient(ctx, c.url)
	if err != nil {
		return c.ExitOnError(err)
	}

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
		resp, err := client.Query(input)
		stop()
		if err != nil {
			c.PrintInteractiveError(err)
			continue
		}

		result := buildQueryResult(resp)
		if len(result.Rows) > 0 {
			table := tablewriter.NewWriter(c.out)
			table.SetAutoFormatHeaders(false)
			table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetAutoWrapText(false)
			for _, row := range result.Rows {
				table.Append(row.Columns)
			}
			table.SetHeader(result.Header)
			table.Render()
			fmt.Fprintf(c.out, "%d points in result\n\n", len(result.Rows))
		} else {
			fmt.Fprintf(c.out, "Empty result\n\n")
		}
	}
}

func (c *Cli) ReadInput(rl *readline.Instance) (string, error) {
	defer rl.SetPrompt(defaultPrompt)

	var input string
	var multiline bool
	for {
		line, err := rl.Readline()
		if err != nil {
			return "", err
		}
		if line == "" {
			continue
		}

		line = strings.TrimSpace(line)

		// multiline
		if strings.HasSuffix(line, `\`) {
			input += strings.TrimSuffix(line, `\`)
			rl.SetPrompt("  -> ")
			multiline = true
			continue
		}

		input += line
		if multiline {
			// Save multi-line input as single-line input into the history
			rl.SaveHistory(input)
		}
		return input, nil
	}
}

func (c *Cli) Exit() int {
	fmt.Fprintln(c.out, "Bye")
	return exitCodeSuccess
}

func (c *Cli) ExitOnError(err error) int {
	fmt.Fprintf(c.out, "ERROR: %s\n", err)
	return exitCodeError
}

func (c *Cli) PrintInteractiveError(err error) {
	fmt.Fprintf(c.out, "ERROR: %s\n", err)
}

func (c *Cli) PrintProgressingMark() func() {
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

type Result struct {
	Header []string
	Rows   []Row
}

type Row struct {
	Columns []string
}

func buildQueryResult(qr *QueryResponse) *Result {
	result := Result{}

	if len(qr.Data.Result) == 0 {
		return &result
	}

	if qr.Data.ResultType == "scalar" {
		// Add header columns.
		result.Header = []string{"timestamp", "value"}

		// Add row.
		timestamp := qr.Data.ResultScalar[0].(float64)
		value := qr.Data.ResultScalar[1].(string)
		result.Rows = []Row{{Columns: []string{formatTimestamp(timestamp), value}}}
		return &result
	}

	if qr.Data.ResultType == "vector" {
		if len(qr.Data.ResultVector) == 0 {
			return &result
		}

		// Add header columns.
		result.Header = []string{"timestamp"}
		firstTimeSeries := qr.Data.ResultVector[0]
		for labelName := range firstTimeSeries.Metric {
			result.Header = append(result.Header, labelName)
		}
		result.Header = append(result.Header, "value")

		// Add rows.
		for _, timeseries := range qr.Data.ResultVector {
			var row Row
			timestamp := timeseries.Point[0].(float64)
			value := timeseries.Point[1].(string)

			row.Columns = append(row.Columns, formatTimestamp(timestamp))
			for _, labelName := range sortedLabelNames(timeseries.Metric) {
				row.Columns = append(row.Columns, timeseries.Metric[labelName])
			}
			row.Columns = append(row.Columns, value)
			result.Rows = append(result.Rows, row)
		}
		return &result
	}

	if qr.Data.ResultType == "matrix" {
		if len(qr.Data.ResultMatrix) == 0 {
			return &result
		}

		// Add header columns.
		result.Header = []string{"timestamp"}
		firstTimeSeries := qr.Data.ResultMatrix[0]
		for labelName := range firstTimeSeries.Metric {
			result.Header = append(result.Header, labelName)
		}
		result.Header = append(result.Header, "value")

		// Add rows.
		for _, timeseries := range qr.Data.ResultMatrix {
			for _, point := range timeseries.Points {
				timestamp := point[0].(float64)
				value := point[1].(string)

				var row Row
				row.Columns = append(row.Columns, formatTimestamp(timestamp))
				for _, labelName := range sortedLabelNames(timeseries.Metric) {
					row.Columns = append(row.Columns, timeseries.Metric[labelName])
				}
				row.Columns = append(row.Columns, value)
				result.Rows = append(result.Rows, row)
			}
		}
		return &result
	}

	// Unreachable.
	return &result
}

func sortedLabelNames(labels map[string]string) []string {
	var labelNames []string
	for l := range labels {
		labelNames = append(labelNames, l)
	}
	sort.Strings(labelNames)
	return labelNames
}

func formatTimestamp(timestamp float64) string {
	t := time.UnixMicro(int64(timestamp * 1_000_000))
	return t.Format(time.RFC3339Nano)
}
