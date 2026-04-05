package generate

type Request struct {
	MainStream     string `json:"mainStream"`
	SubStream      string `json:"subStream,omitempty"`
	Name           string `json:"name,omitempty"`
	ExistingConfig string `json:"existingConfig,omitempty"`

	Go2RTC  *Go2RTCOverride  `json:"go2rtc,omitempty"`
	Frigate *FrigateOverride  `json:"frigate,omitempty"`

	Objects   []string      `json:"objects,omitempty"`
	Record    *RecordConfig `json:"record,omitempty"`
	Detect    *DetectConfig `json:"detect,omitempty"`
	Snapshots *BoolConfig   `json:"snapshots,omitempty"`
	Motion    *MotionConfig `json:"motion,omitempty"`

	FFmpeg        *FFmpegConfig   `json:"ffmpeg,omitempty"`
	Live          *LiveConfig     `json:"live,omitempty"`
	Audio         *AudioConfig    `json:"audio,omitempty"`
	Birdseye      *BirdseyeConfig `json:"birdseye,omitempty"`
	ONVIF         *ONVIFConfig    `json:"onvif,omitempty"`
	PTZ           *PTZConfig      `json:"ptz,omitempty"`
	Notifications *BoolConfig     `json:"notifications,omitempty"`
	UI            *UIConfig       `json:"ui,omitempty"`
}

type Go2RTCOverride struct {
	MainStreamName   string `json:"mainStreamName,omitempty"`
	SubStreamName    string `json:"subStreamName,omitempty"`
	MainStreamSource string `json:"mainStreamSource,omitempty"`
	SubStreamSource  string `json:"subStreamSource,omitempty"`
}

type FrigateOverride struct {
	MainStreamPath      string `json:"mainStreamPath,omitempty"`
	SubStreamPath       string `json:"subStreamPath,omitempty"`
	MainStreamInputArgs string `json:"mainStreamInputArgs,omitempty"`
	SubStreamInputArgs  string `json:"subStreamInputArgs,omitempty"`
}

type RecordConfig struct {
	Enabled       bool    `json:"enabled"`
	RetainDays    float64 `json:"retain_days,omitempty"`
	Mode          string  `json:"mode,omitempty"`
	AlertsDays    float64 `json:"alerts_days,omitempty"`
	DetectionDays float64 `json:"detections_days,omitempty"`
	PreCapture    int     `json:"pre_capture,omitempty"`
	PostCapture   int     `json:"post_capture,omitempty"`
}

type DetectConfig struct {
	Enabled bool `json:"enabled"`
	FPS     int  `json:"fps,omitempty"`
	Width   int  `json:"width,omitempty"`
	Height  int  `json:"height,omitempty"`
}

type MotionConfig struct {
	Enabled     bool `json:"enabled"`
	Threshold   int  `json:"threshold,omitempty"`
	ContourArea int  `json:"contour_area,omitempty"`
}

type FFmpegConfig struct {
	HWAccel string `json:"hwaccel,omitempty"`
	GPU     int    `json:"gpu,omitempty"`
}

type LiveConfig struct {
	Height  int `json:"height,omitempty"`
	Quality int `json:"quality,omitempty"`
}

type AudioConfig struct {
	Enabled bool     `json:"enabled"`
	Filters []string `json:"filters,omitempty"`
}

type BirdseyeConfig struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode,omitempty"`
}

type ONVIFConfig struct {
	Host          string   `json:"host,omitempty"`
	Port          int      `json:"port,omitempty"`
	User          string   `json:"user,omitempty"`
	Password      string   `json:"password,omitempty"`
	AutoTracking  bool     `json:"autotracking,omitempty"`
	RequiredZones []string `json:"required_zones,omitempty"`
}

type PTZConfig struct {
	Enabled bool              `json:"enabled"`
	Presets map[string]string `json:"presets,omitempty"`
}

type BoolConfig struct {
	Enabled bool `json:"enabled"`
}

type UIConfig struct {
	Order     int  `json:"order,omitempty"`
	Dashboard bool `json:"dashboard"`
}

type Response struct {
	Config string `json:"config"`
	Added  []int  `json:"added"` // 1-based line numbers of added lines
}
