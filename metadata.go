package wesplot

type ChartOptions struct {
	Title  string
	XLabel string   `json:",omitempty"`
	YLabel string   `json:",omitempty"`
	YMin   *float64 `json:",omitempty"`
	YMax   *float64 `json:",omitempty"`
}

type Metadata struct {
	WindowSize   int
	Columns      []string
	XIsTimestamp bool
	YUnit        string
	ChartOptions ChartOptions
}
