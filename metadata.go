package wesplot

type EChartsOptionXAxis struct {
	Name string `json:"name"`
}
type EChartsOptionYAxis struct {
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
	Name string   `json:"name"`
}

type EChartsOptionTitle struct {
	Text string `json:"text"`
}

type EChartsOption struct {
	Title EChartsOptionTitle `json:"title"`
	XAxis EChartsOptionXAxis `json:"xAxis"`
	YAxis EChartsOptionYAxis `json:"yAxis"`
}

type Metadata struct {
	WindowSize    int
	Columns       []string
	YUnit         string
	EChartsOption EChartsOption
}
