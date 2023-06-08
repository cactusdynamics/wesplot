package wesplot

type WesplotOptions struct {
	Title    string
	Columns  []string
	XLabel   string
	YLabel   string
	YMin     *float64 `json:",omitempty"`
	YMax     *float64 `json:",omitempty"`
	YUnit    string
	ShowLine bool
}

type Metadata struct {
	WindowSize     int
	XIsTimestamp   bool
	RelativeStart  bool
	WesplotOptions WesplotOptions
}
