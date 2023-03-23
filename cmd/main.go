package main

import (
	"context"
	"os"

	"github.com/cactusdynamics/wesplot"
)

func main() {
	metadata := wesplot.Metadata{
		RollingWindowSize: 10000,
		EChartsOption: wesplot.EChartsOption{
			Title: wesplot.Title{
				Text: "Plot",
			},
		},
	}

	dataSource := wesplot.NewCsvDataSource(os.Stdin, []wesplot.Operator{}, -1, []string{"data"})
	dataBroadcaster := wesplot.NewDataBroadcaster(dataSource, 10000)
	server := wesplot.NewHttpServer(dataBroadcaster, "0.0.0.0:8080", metadata)

	dataBroadcaster.Start(context.Background())
	server.Run()
}
