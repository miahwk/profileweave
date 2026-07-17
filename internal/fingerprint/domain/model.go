package domain

type Screen struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	DPR    float64 `json:"dpr"`
}

type Fingerprint struct {
	OS           string   `json:"os"`
	UAMode       string   `json:"uaMode"`
	UserAgent    string   `json:"userAgent,omitempty"`
	Locale       string   `json:"locale"`
	Languages    []string `json:"languages"`
	Timezone     string   `json:"timezone"`
	Screen       Screen   `json:"screen"`
	CPUCores     int      `json:"cpuCores"`
	MemoryGB     int      `json:"memoryGB"`
	WebRTCPolicy string   `json:"webrtcPolicy"`
}

type Proxy struct {
	Mode string `json:"mode"`
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
}

func Default() Fingerprint {
	return Fingerprint{
		OS: "native", UAMode: "native", Locale: "en-US",
		Languages: []string{"en-US", "en"}, Timezone: "UTC",
		Screen:   Screen{Width: 1920, Height: 1080, DPR: 1},
		CPUCores: 4, MemoryGB: 8, WebRTCPolicy: "native",
	}
}
