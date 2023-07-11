package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cactusdynamics/wesplot"
	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

var options struct {
	Host    string `short:"h" long:"host" default:"0.0.0.0" description:"the IP to start the server on. Default to 0.0.0.0 (all interfaces)"`
	Port    uint16 `short:"p" long:"port" default:"5274"`
	Verbose bool   `short:"v" long:"verbose" description:"Show debug logs"`
	Tee     bool   `short:"T" long:"tee" description:"Write the data (and generated timestamp if applicable) in a CSV into stdout in addition to visualizing with the plot"`

	Title     string   `short:"t" long:"title" default:"Wesplot" description:"Title of the plot. Defaults to 'Plot'"`
	YMin      *float64 `short:"m" long:"ymin" description:"The minimum value for y (default: auto scaling)"`
	YMax      *float64 `short:"M" long:"ymax" description:"The max value for y (default: auto scaling)"`
	YUnit     string   `short:"u" long:"yunit" description:"The unit for the Y axis"`
	XLabel    string   `long:"xlabel" description:"Label for the X axis"`
	YLabel    string   `long:"ylabel" description:"Label for the Y axis"`
	ChartType string   `long:"chart-type" choice:"scatter" choice:"line" default:"line" description:"The type of chart to plot (scatter or line). Defaults to 'line'"`

	XIndex        int  `short:"x" long:"xindex" default:"-1" description:"The index for the x column. If not specified, the x value is generated as the receive timestamp. If specified, this is will let the front end know the x value is not a timestamp. Mutually exclusive with --tindex."`
	TIndex        int  `long:"tindex" default:"-1" description:"The index for the timestamp column. If not specified, the x value is generated as the receive timestamp. Mutually exclusive with --xindex."`
	RelativeStart bool `short:"s" long:"relative-start" description:"If this is specified, the X values will be normalized by the first value. i.e x_i = x_original_i - x_0. Applies to both timestamps and non timestamps."`

	NumColumns int      `short:"n" long:"num-columns" description:"The number of columns expected for the input data. If specified, input data rows with different number of columns will be ignored."`
	Columns    []string `short:"c" long:"columns" description:"The columns labels for the input data. This option supercedes num-columns and will also be used to validate the input data like --num-columns."`

	WindowSize    int           `short:"w" long:"window-size" default:"1800" description:"the number of data rows cached on a rolling windows basis. default: 1800 which means 1800 data points will be cached by the tool and sent any time the browser connects"`
	FlushInterval time.Duration `long:"flush-interval" default:"250ms" description:"the flush interval dictates how long the backend waits before flushing the data to the frontend. If the frontend is too slow and cannot keep up updating the plot, increase this number"`

	xIsTimestamp bool
}

func parseOptions() {
	_, err := flags.ParseArgs(&options, os.Args)
	if err != nil {
		if flags.WroteHelp(err) {
			os.Exit(0)
		}
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
		}
	}

	if options.YMin != nil && options.YMax != nil {
		if *options.YMin >= *options.YMax {
			logrus.Errorf("YMax (%f) must be greater than YMin (%f)", *options.YMax, *options.YMin)
			os.Exit(1)
		}
	}

	// TODO: this code is kind of funky but OK.
	if options.XIndex != -1 {
		if options.TIndex != -1 {
			logrus.Error("both --xindex and --tindex is specified and this is mutually exclusive")
			os.Exit(1)
		}

		options.xIsTimestamp = false
	} else {
		if options.TIndex >= 0 {
			options.XIndex = options.TIndex // lol this is not good but it works for now.
		}
		options.xIsTimestamp = true
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
		WindowSize:    options.WindowSize,
		XIsTimestamp:  options.xIsTimestamp,
		RelativeStart: options.RelativeStart,
		WesplotOptions: wesplot.WesplotOptions{
			Title:     options.Title,
			Columns:   options.Columns, // TODO: dynamic columns
			XLabel:    options.XLabel,
			YLabel:    options.YLabel,
			YMin:      options.YMin,
			YMax:      options.YMax,
			YUnit:     options.YUnit,
			ChartType: options.ChartType,
		},
	}

	var stringReader wesplot.StringReader = wesplot.NewRelaxedStringReader(os.Stdin)
	var dataRowReader wesplot.DataRowReader = &wesplot.TextToDataRowReader{
		Input:                  stringReader,
		XIndex:                 options.XIndex,
		Columns:                options.Columns,
		ExpectExactColumnCount: true, // Not sure how to deal with dynamic columns so for now we need exact column count
	}

	dataBroadcaster := wesplot.NewDataBroadcaster(dataRowReader, options.WindowSize, options.Tee)
	server := wesplot.NewHttpServer(dataBroadcaster, options.Host, options.Port, metadata, options.FlushInterval)

	dataBroadcaster.Start(context.Background())
	server.Run()
}
