package wesplot

type ChartOptions struct {
	Title  string
	XLabel string
	YLabel string
	YMin   *float64
	YMax   *float64
}

type Metadata struct {
	WindowSize   int
	Columns      []string
	YUnit        string
	ChartOptions ChartOptions
}
