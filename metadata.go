package wesplot

type Title struct {
	Text string `json:"text"`
}

type EChartOptions struct {
	Title Title `json:"title"`
}

type Metadata struct {
	RollingWindowSize int
	EChartOptions     EChartOptions
}
