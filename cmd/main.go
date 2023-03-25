package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cactusdynamics/wesplot"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

var options struct {
	Verbose bool `short:"v" long:"verbose" description:"Show debug logs"`

	Title string   `short:"t" long:"title" description:"Title of the plot"`
	YMin  *float64 `short:"M" long:"ymin" description:"The minimum value for y (default: auto scaling)"`
	YMax  *float64 `short:"m" long:"ymax" description:"The max value for y (default: auto scaling)"`
	YUnit string   `short:"u" long:"yunit" description:"The unit for the Y axis"`

	NumColumns int      `short:"n" long:"num-columns" description:"The number of columns expected for the input data. If specified, input data rows with different number of columns will be ignored."`
	Columns    []string `short:"c" long:"columns" description:"The columns labels for the input data. This option supercedes num-columns and will also be used to validate the input data like --num-columns."`

	WindowSize int `short:"w" long:"window-size" default:"1000" description:"the number of data rows cached on a rolling windows basis. default: 1000 which means 1000 data points will be cached by the tool and sent any time the browser connects"`

	// This is set to true neither NumColumns nor Columns is specified.
	// Current unused.
	dynamicColumnsCount bool
}

func parseOptions() {
	_, err := flags.ParseArgs(&options, os.Args)
	if err != nil {
		panic(err)
	}

	if options.NumColumns > 0 {
		if len(options.Columns) == 0 {
			// User specified --num-columns but not --columns, so we construct it
			// artificially with y1, y2, y3, ... and so on.
			for i := 0; i < options.NumColumns; i++ {
				options.Columns = append(options.Columns, fmt.Sprintf("y%d", i))
			}
		} else {
			// The user specifies both. This is redundant and unnecessary (and could
			// be conflicting if len(columns) != num-columns), so we just the
			// --columns is the source of truth.
			logrus.Warn("both --columns and --num-columns are specified. --num-columns is thus ignored.")
		}
	} else {
		if len(options.Columns) == 0 {
			// This happens when the user specifies neither --num-columns nor
			// --columns. For now, we assume a single column. Later on, we can make it
			// dynamic.
			options.Columns = []string{"y1"}
			options.dynamicColumnsCount = true
		}
	}

	if options.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("logging verbose output")
		logrus.Debug("options:")
		data, err := json.MarshalIndent(options, "", "  ")
		if err != nil {
			panic(err)
		}

		fmt.Println(string(data))
	}
}

func main() {
	parseOptions()

	logrus.Infof("starting wesplot %v", wesplot.Version)

	metadata := wesplot.Metadata{
		WindowSize: options.WindowSize,
		Columns:    options.Columns, // TODO: dynamic columns
		YUnit:      options.YUnit,
		ChartOptions: wesplot.ChartOptions{
			Title:  "Plot title",
			XLabel: "X label",
			YLabel: "Y label",
			YMin:   options.YMin,
			YMax:   options.YMax,
		},
	}

	dataSource := wesplot.NewCsvDataSource(os.Stdin, []wesplot.Operator{}, -1, options.Columns, options.dynamicColumnsCount)
	dataBroadcaster := wesplot.NewDataBroadcaster(dataSource, options.WindowSize)
	server := wesplot.NewHttpServer(dataBroadcaster, "0.0.0.0:8080", metadata)

	dataBroadcaster.Start(context.Background())
	server.Run()
}
