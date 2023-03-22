package main

import (
	"context"
	"os"

	"github.com/cactusdynamics/wesplot"
)

func main() {
	dataSource := wesplot.NewCsvDataSource(os.Stdin, []wesplot.Operator{}, -1, []string{"data"})
	dataBroadcaster := wesplot.NewDataBroadcaster(dataSource, 10000)
	server := wesplot.NewHttpServer(dataBroadcaster, "0.0.0.0:8080")

	dataBroadcaster.Start(context.Background())
	server.Run()
}
