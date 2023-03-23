package wesplot

type Title struct {
	Text string `json:"text"`
}

type EChartsOption struct {
	Title Title `json:"title"`
}

type Metadata struct {
	RollingWindowSize int
	EChartsOption     EChartsOption
}
