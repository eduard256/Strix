package generate

import (
	"strings"
	"testing"
)

// End-to-end tests for writer.go: every writeX function is exercised through
// the public Generate entry-point so the tests survive internal refactoring.
// Shared helpers (mustGen / assertContains / assertNotContains / countOccurrences)
// are defined in xiaomi_test.go.

// baseRTSP is a neutral main-stream URL that does not trigger any extractor
// (registry), does not trigger needMP4, and has a stable IP for name derivation.
const baseRTSP = "rtsp://admin:pw@10.0.20.10:554/Streaming/Channels/101"
const baseSubRTSP = "rtsp://admin:pw@10.0.20.10:554/Streaming/Channels/102"

// --- writeInput (roles) -------------------------------------------------------

// Without sub: a single input carries both detect and record roles.
func TestWriter_Input_SingleRoleCombined(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})

	assertContains(t, cfg, "      inputs:\n")
	assertContains(t, cfg, "        - path: rtsp://127.0.0.1:8554/10_0_20_10_main\n")
	assertContains(t, cfg, "          input_args: preset-rtsp-restream\n")
	assertContains(t, cfg, "          roles:\n            - detect\n            - record\n")

	if n := countOccurrences(cfg, "- path:"); n != 1 {
		t.Errorf("expected 1 input, got %d\n%s", n, cfg)
	}
}

// With sub: sub gets detect, main gets record -- sub must come FIRST in inputs list.
func TestWriter_Input_SubGetsDetectMainGetsRecord(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		SubStream:  baseSubRTSP,
	})

	assertContains(t, cfg, "        - path: rtsp://127.0.0.1:8554/10_0_20_10_sub\n")
	assertContains(t, cfg, "        - path: rtsp://127.0.0.1:8554/10_0_20_10_main\n")

	subIdx := strings.Index(cfg, "rtsp://127.0.0.1:8554/10_0_20_10_sub")
	mainIdx := strings.Index(cfg, "rtsp://127.0.0.1:8554/10_0_20_10_main\n          input_args")
	if !(subIdx > 0 && mainIdx > 0 && subIdx < mainIdx) {
		t.Errorf("expected sub path to appear before main path in inputs:\n%s", cfg)
	}

	// sub has only detect role, main has only record
	detectBlock := "        - path: rtsp://127.0.0.1:8554/10_0_20_10_sub\n" +
		"          input_args: preset-rtsp-restream\n" +
		"          roles:\n            - detect\n"
	recordBlock := "        - path: rtsp://127.0.0.1:8554/10_0_20_10_main\n" +
		"          input_args: preset-rtsp-restream\n" +
		"          roles:\n            - record\n"
	assertContains(t, cfg, detectBlock)
	assertContains(t, cfg, recordBlock)
}

// --- needMP4 ------------------------------------------------------------------

// bubble:// MUST produce a restream path ending in ?mp4 -- Frigate bubble bug.
func TestWriter_NeedMP4_Bubble(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: "bubble://admin:pw@10.0.20.50:80/bubble/live?ch=0&stream=0",
	})
	assertContains(t, cfg, "- path: rtsp://127.0.0.1:8554/10_0_20_50_main?mp4\n")
}

// Same for sub stream.
func TestWriter_NeedMP4_BubbleSub(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		SubStream:  "bubble://admin:pw@10.0.20.10:80/bubble/live?ch=0&stream=1",
	})
	assertContains(t, cfg, "- path: rtsp://127.0.0.1:8554/10_0_20_10_sub?mp4\n")
	// main is rtsp, MUST NOT have ?mp4
	assertContains(t, cfg, "- path: rtsp://127.0.0.1:8554/10_0_20_10_main\n")
	assertNotContains(t, cfg, "10_0_20_10_main?mp4")
}

// rtsp:// and other non-listed schemes MUST NOT append ?mp4.
func TestWriter_NeedMP4_RTSPNotAppended(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "?mp4")
}

// --- writeFFmpegGlobal --------------------------------------------------------

// FFmpeg == nil -> no hwaccel_args, no gpu.
func TestWriter_FFmpeg_Nil(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "hwaccel_args:")
	assertNotContains(t, cfg, "gpu:")
}

// HWAccel="auto" is a sentinel -- it means "let Frigate decide", don't emit.
func TestWriter_FFmpeg_HWAccelAutoSkipped(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		FFmpeg:     &FFmpegConfig{HWAccel: "auto"},
	})
	assertNotContains(t, cfg, "hwaccel_args:")
}

// Explicit preset is written verbatim.
func TestWriter_FFmpeg_HWAccelExplicit(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		FFmpeg:     &FFmpegConfig{HWAccel: "preset-vaapi", GPU: 1},
	})
	assertContains(t, cfg, "      hwaccel_args: preset-vaapi\n")
	assertContains(t, cfg, "      gpu: 1\n")
}

// GPU 0 (default) must not be written; >0 is.
func TestWriter_FFmpeg_GPUZeroSkipped(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		FFmpeg:     &FFmpegConfig{HWAccel: "preset-vaapi"},
	})
	assertContains(t, cfg, "hwaccel_args: preset-vaapi")
	assertNotContains(t, cfg, "gpu:")
}

// --- writeLive ----------------------------------------------------------------

// No sub + no Live config -> no live: block at all.
func TestWriter_Live_NoSubNoLive_Absent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    live:\n")
}

// Sub present -> live.streams with Main + Sub stream labels.
func TestWriter_Live_WithSub_StreamsMap(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		SubStream:  baseSubRTSP,
	})
	assertContains(t, cfg, "    live:\n      streams:\n")
	assertContains(t, cfg, "        Main Stream: 10_0_20_10_main\n")
	assertContains(t, cfg, "        Sub Stream: 10_0_20_10_sub\n")
}

// Live.Height / Live.Quality override defaults.
func TestWriter_Live_HeightAndQuality(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Live:       &LiveConfig{Height: 720, Quality: 6},
	})
	assertContains(t, cfg, "    live:\n")
	assertContains(t, cfg, "      height: 720\n")
	assertContains(t, cfg, "      quality: 6\n")
}

// Live with zero values omits those fields.
func TestWriter_Live_ZeroValuesOmitted(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Live:       &LiveConfig{},
	})
	assertNotContains(t, cfg, "      height: 0\n")
	assertNotContains(t, cfg, "      quality: 0\n")
}

// --- writeDetect --------------------------------------------------------------

// Detect == nil -> default is enabled: true (Frigate needs explicit detect block).
func TestWriter_Detect_NilDefaultsToEnabled(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertContains(t, cfg, "    detect:\n      enabled: true\n")
}

// Explicit enabled: false is written.
func TestWriter_Detect_ExplicitDisabled(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Detect:     &DetectConfig{Enabled: false},
	})
	assertContains(t, cfg, "    detect:\n      enabled: false\n")
}

// FPS/Width/Height > 0 are written.
func TestWriter_Detect_CustomFPSWidthHeight(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Detect:     &DetectConfig{Enabled: true, FPS: 10, Width: 1920, Height: 1080},
	})
	assertContains(t, cfg, "    detect:\n      enabled: true\n")
	assertContains(t, cfg, "      fps: 10\n")
	assertContains(t, cfg, "      width: 1920\n")
	assertContains(t, cfg, "      height: 1080\n")
}

// Zero values inside Detect are omitted (not written as "fps: 0" etc).
func TestWriter_Detect_ZeroValuesOmitted(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Detect:     &DetectConfig{Enabled: true},
	})
	assertNotContains(t, cfg, "      fps: 0\n")
	assertNotContains(t, cfg, "      width: 0\n")
	assertNotContains(t, cfg, "      height: 0\n")
}

// Setting Objects auto-enables detect even if Detect was nil (see Generate).
func TestWriter_Detect_ObjectsAutoEnable(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Objects:    []string{"car"},
	})
	assertContains(t, cfg, "    detect:\n      enabled: true\n")
	assertContains(t, cfg, "    objects:\n      track:\n        - car\n")
}

// Objects + Detect{Enabled:false} -> Generate flips Enabled to true.
func TestWriter_Detect_ObjectsOverridesDisabledDetect(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Objects:    []string{"dog"},
		Detect:     &DetectConfig{Enabled: false},
	})
	assertContains(t, cfg, "    detect:\n      enabled: true\n")
}

// --- writeObjects -------------------------------------------------------------

// Empty Objects list -> default ["person"].
func TestWriter_Objects_DefaultPerson(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertContains(t, cfg, "    objects:\n      track:\n        - person\n")
}

// Explicit list preserves order.
func TestWriter_Objects_ExplicitListPreservesOrder(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Objects:    []string{"car", "person", "cat"},
	})
	assertContains(t, cfg, "    objects:\n      track:\n        - car\n        - person\n        - cat\n")
}

// --- writeMotion --------------------------------------------------------------

// Motion == nil -> block absent.
func TestWriter_Motion_NilAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    motion:\n")
}

// Motion{} -> block with enabled: false.
func TestWriter_Motion_DisabledStillWritesBlock(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Motion:     &MotionConfig{Enabled: false},
	})
	assertContains(t, cfg, "    motion:\n      enabled: false\n")
}

// Threshold / ContourArea > 0 are written.
func TestWriter_Motion_WithThresholdAndContour(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Motion:     &MotionConfig{Enabled: true, Threshold: 30, ContourArea: 10},
	})
	assertContains(t, cfg, "    motion:\n      enabled: true\n")
	assertContains(t, cfg, "      threshold: 30\n")
	assertContains(t, cfg, "      contour_area: 10\n")
}

// Zero values inside Motion are omitted.
func TestWriter_Motion_ZeroOmitted(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Motion:     &MotionConfig{Enabled: true},
	})
	assertNotContains(t, cfg, "threshold: 0")
	assertNotContains(t, cfg, "contour_area: 0")
}

// --- writeRecord --------------------------------------------------------------

// Record == nil -> default enabled: true.
func TestWriter_Record_NilDefaultEnabled(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	// per-camera record block (top-level "record:\n  enabled: true" also exists -- that's global)
	assertContains(t, cfg, "    record:\n      enabled: true\n")
}

// Record.Enabled: false is written.
func TestWriter_Record_Disabled(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record:     &RecordConfig{Enabled: false},
	})
	assertContains(t, cfg, "    record:\n      enabled: false\n")
}

// Only retain days -> retain block with days only.
func TestWriter_Record_RetainDaysOnly(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record:     &RecordConfig{Enabled: true, RetainDays: 7},
	})
	assertContains(t, cfg, "    record:\n      enabled: true\n      retain:\n        days: 7\n")
	assertNotContains(t, cfg, "        mode:")
}

// Retain mode only (no days).
func TestWriter_Record_RetainModeOnly(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record:     &RecordConfig{Enabled: true, Mode: "motion"},
	})
	assertContains(t, cfg, "      retain:\n        mode: motion\n")
	assertNotContains(t, cfg, "        days:")
}

// Fractional retain days -> %g formatting (0.5, not 5e-01).
func TestWriter_Record_FractionalRetainDays(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record:     &RecordConfig{Enabled: true, RetainDays: 0.5},
	})
	assertContains(t, cfg, "        days: 0.5\n")
}

// Alerts block: AlertsDays + PreCapture + PostCapture.
func TestWriter_Record_AlertsBlock(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record:     &RecordConfig{Enabled: true, AlertsDays: 14, PreCapture: 5, PostCapture: 10},
	})
	assertContains(t, cfg, "      alerts:\n")
	assertContains(t, cfg, "        retain:\n          days: 14\n")
	assertContains(t, cfg, "        pre_capture: 5\n")
	assertContains(t, cfg, "        post_capture: 10\n")
}

// Only PreCapture -> alerts block appears with only pre_capture.
func TestWriter_Record_OnlyPreCaptureStillEmitsAlerts(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record:     &RecordConfig{Enabled: true, PreCapture: 5},
	})
	assertContains(t, cfg, "      alerts:\n")
	assertContains(t, cfg, "        pre_capture: 5\n")
	assertNotContains(t, cfg, "          days:")
}

// DetectionDays writes a separate detections block.
func TestWriter_Record_DetectionsDays(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record:     &RecordConfig{Enabled: true, DetectionDays: 30},
	})
	assertContains(t, cfg, "      detections:\n        retain:\n          days: 30\n")
}

// All fields combined -- retain, alerts, detections all present.
func TestWriter_Record_AllFieldsCombined(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Record: &RecordConfig{
			Enabled: true, RetainDays: 7, Mode: "all",
			AlertsDays: 14, PreCapture: 5, PostCapture: 10,
			DetectionDays: 30,
		},
	})
	assertContains(t, cfg, "      retain:\n        days: 7\n        mode: all\n")
	assertContains(t, cfg, "      alerts:\n        retain:\n          days: 14\n        pre_capture: 5\n        post_capture: 10\n")
	assertContains(t, cfg, "      detections:\n        retain:\n          days: 30\n")
}

// --- writeSnapshots -----------------------------------------------------------

func TestWriter_Snapshots_NilAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    snapshots:\n")
}

func TestWriter_Snapshots_DisabledAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Snapshots:  &BoolConfig{Enabled: false},
	})
	assertNotContains(t, cfg, "    snapshots:\n")
}

func TestWriter_Snapshots_Enabled(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Snapshots:  &BoolConfig{Enabled: true},
	})
	assertContains(t, cfg, "    snapshots:\n      enabled: true\n")
}

// --- writeAudio ---------------------------------------------------------------

func TestWriter_Audio_NilAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    audio:\n")
}

func TestWriter_Audio_DisabledAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Audio:      &AudioConfig{Enabled: false},
	})
	assertNotContains(t, cfg, "    audio:\n")
}

func TestWriter_Audio_EnabledNoFilters(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Audio:      &AudioConfig{Enabled: true},
	})
	assertContains(t, cfg, "    audio:\n      enabled: true\n")
	assertNotContains(t, cfg, "      filters:\n")
}

func TestWriter_Audio_WithFilters(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Audio:      &AudioConfig{Enabled: true, Filters: []string{"speech", "bark"}},
	})
	assertContains(t, cfg, "      filters:\n        - speech\n        - bark\n")
}

// --- writeBirdseye ------------------------------------------------------------

func TestWriter_Birdseye_NilAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    birdseye:\n")
}

// Enabled:false is still written (birdseye may be globally on, per-camera off).
func TestWriter_Birdseye_DisabledExplicit(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Birdseye:   &BirdseyeConfig{Enabled: false},
	})
	assertContains(t, cfg, "    birdseye:\n      enabled: false\n")
}

func TestWriter_Birdseye_EnabledWithMode(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Birdseye:   &BirdseyeConfig{Enabled: true, Mode: "motion"},
	})
	assertContains(t, cfg, "    birdseye:\n      enabled: true\n      mode: motion\n")
}

func TestWriter_Birdseye_EmptyModeOmitted(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Birdseye:   &BirdseyeConfig{Enabled: true},
	})
	assertContains(t, cfg, "    birdseye:\n      enabled: true\n")
	assertNotContains(t, cfg, "      mode:")
}

// --- writeONVIF ---------------------------------------------------------------

// THIS IS THE BUG THE USER JUST FIXED: no ONVIF in request -> no block in config.
// (host comes from the frontend; if empty, block must not appear.)
func TestWriter_ONVIF_NilAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    onvif:\n")
}

// Empty host -> block skipped even if ONVIFConfig is non-nil.
func TestWriter_ONVIF_EmptyHostAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{User: "admin"},
	})
	assertNotContains(t, cfg, "    onvif:\n")
}

// Host without port -> default port 80.
func TestWriter_ONVIF_DefaultPort80(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10"},
	})
	assertContains(t, cfg, "    onvif:\n      host: 10.0.20.10\n      port: 80\n")
}

// Explicit port overrides default.
func TestWriter_ONVIF_ExplicitPort(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10", Port: 2020},
	})
	assertContains(t, cfg, "      port: 2020\n")
	assertNotContains(t, cfg, "      port: 80\n")
}

// User set -> user + password lines (password lands even if empty -- by design).
func TestWriter_ONVIF_UserPassword(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10", User: "admin", Password: "s3cret"},
	})
	assertContains(t, cfg, "      user: admin\n")
	assertContains(t, cfg, "      password: s3cret\n")
}

// No user -> no user/password lines.
func TestWriter_ONVIF_NoUserNoCredentials(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10"},
	})
	assertNotContains(t, cfg, "      user:")
	assertNotContains(t, cfg, "      password:")
}

// Autotracking enabled -> autotracking.enabled: true.
func TestWriter_ONVIF_Autotracking(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10", AutoTracking: true},
	})
	assertContains(t, cfg, "      autotracking:\n        enabled: true\n")
}

// Autotracking + required_zones -> nested list.
func TestWriter_ONVIF_AutotrackingRequiredZones(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF: &ONVIFConfig{
			Host: "10.0.20.10", AutoTracking: true,
			RequiredZones: []string{"driveway", "yard"},
		},
	})
	assertContains(t, cfg, "        required_zones:\n          - driveway\n          - yard\n")
}

// required_zones without autotracking -> NOT written.
func TestWriter_ONVIF_RequiredZonesWithoutAutotracking(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF: &ONVIFConfig{
			Host:          "10.0.20.10",
			RequiredZones: []string{"driveway"},
		},
	})
	assertNotContains(t, cfg, "required_zones:")
}

// --- writePTZ (only written inside onvif block) -------------------------------

// PTZ without ONVIF -> nothing written (writer is nested inside writeONVIF).
func TestWriter_PTZ_WithoutONVIFNotWritten(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		PTZ:        &PTZConfig{Enabled: true, Presets: map[string]string{"home": "TOKEN1"}},
	})
	assertNotContains(t, cfg, "    onvif:\n")
	assertNotContains(t, cfg, "        presets:\n")
}

// PTZ with ONVIF -> ptz.presets nested under onvif.
func TestWriter_PTZ_WithONVIF(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10"},
		PTZ:        &PTZConfig{Enabled: true, Presets: map[string]string{"home": "TOKEN1"}},
	})
	assertContains(t, cfg, "      ptz:\n        presets:\n          home: TOKEN1\n")
}

// Empty PTZ.Presets -> no ptz block even if ONVIF present.
func TestWriter_PTZ_EmptyPresetsNoBlock(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10"},
		PTZ:        &PTZConfig{Enabled: true},
	})
	assertNotContains(t, cfg, "      ptz:")
}

// --- writeNotifications -------------------------------------------------------

func TestWriter_Notifications_NilAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    notifications:\n")
}

func TestWriter_Notifications_DisabledAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream:    baseRTSP,
		Notifications: &BoolConfig{Enabled: false},
	})
	assertNotContains(t, cfg, "    notifications:\n")
}

func TestWriter_Notifications_Enabled(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream:    baseRTSP,
		Notifications: &BoolConfig{Enabled: true},
	})
	assertContains(t, cfg, "    notifications:\n      enabled: true\n")
}

// --- writeUI ------------------------------------------------------------------

func TestWriter_UI_NilAbsent(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})
	assertNotContains(t, cfg, "    ui:\n")
}

// Dashboard:true is the default -> block emitted (because req.UI != nil) but no `dashboard: false`.
func TestWriter_UI_DashboardTrueDefault(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		UI:         &UIConfig{Dashboard: true},
	})
	assertContains(t, cfg, "    ui:\n")
	assertNotContains(t, cfg, "      dashboard:")
}

// Dashboard:false is written (hide from dashboard).
func TestWriter_UI_DashboardFalse(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		UI:         &UIConfig{Dashboard: false},
	})
	assertContains(t, cfg, "    ui:\n      dashboard: false\n")
}

// Order > 0 written; Order 0 skipped.
func TestWriter_UI_OrderWritten(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		UI:         &UIConfig{Order: 5, Dashboard: true},
	})
	assertContains(t, cfg, "      order: 5\n")
}

func TestWriter_UI_OrderZeroSkipped(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		UI:         &UIConfig{Dashboard: true},
	})
	assertNotContains(t, cfg, "      order:")
}

// --- Frigate overrides --------------------------------------------------------

func TestWriter_FrigateOverride_MainStreamPath(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Frigate: &FrigateOverride{
			MainStreamPath: "rtsp://10.0.0.5:8554/custom_main",
		},
	})
	assertContains(t, cfg, "- path: rtsp://10.0.0.5:8554/custom_main\n")
	assertNotContains(t, cfg, "- path: rtsp://127.0.0.1:8554/10_0_20_10_main\n")
}

func TestWriter_FrigateOverride_MainStreamInputArgs(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Frigate:    &FrigateOverride{MainStreamInputArgs: "-rtsp_transport tcp -timeout 5000000"},
	})
	assertContains(t, cfg, "          input_args: -rtsp_transport tcp -timeout 5000000\n")
	assertNotContains(t, cfg, "          input_args: preset-rtsp-restream\n")
}

func TestWriter_FrigateOverride_SubStreamPathAndArgs(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		SubStream:  baseSubRTSP,
		Frigate: &FrigateOverride{
			SubStreamPath:      "rtsp://10.0.0.5:8554/custom_sub",
			SubStreamInputArgs: "preset-rtsp-udp",
		},
	})
	assertContains(t, cfg, "- path: rtsp://10.0.0.5:8554/custom_sub\n")
	assertContains(t, cfg, "          input_args: preset-rtsp-udp\n")
}

// --- Go2RTC overrides ---------------------------------------------------------

func TestWriter_Go2RTCOverride_MainStreamName(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Go2RTC:     &Go2RTCOverride{MainStreamName: "front_door"},
	})
	assertContains(t, cfg, "    'front_door':\n      - rtsp://admin:pw@10.0.20.10:554/Streaming/Channels/101\n")
	// Frigate input path must follow the renamed stream.
	assertContains(t, cfg, "- path: rtsp://127.0.0.1:8554/front_door\n")
}

func TestWriter_Go2RTCOverride_MainStreamSource(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		Go2RTC:     &Go2RTCOverride{MainStreamSource: "ffmpeg:file.mp4#video=h264"},
	})
	assertContains(t, cfg, "      - ffmpeg:file.mp4#video=h264\n")
	assertNotContains(t, cfg, "      - rtsp://admin:pw@10.0.20.10:554/Streaming/Channels/101\n")
}

func TestWriter_Go2RTCOverride_SubStreamName(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		SubStream:  baseSubRTSP,
		Go2RTC:     &Go2RTCOverride{SubStreamName: "front_door_low"},
	})
	assertContains(t, cfg, "    'front_door_low':\n")
	assertContains(t, cfg, "- path: rtsp://127.0.0.1:8554/front_door_low\n")
	// live.streams must use the renamed sub
	assertContains(t, cfg, "        Sub Stream: front_door_low\n")
}

// --- Name override ------------------------------------------------------------

func TestWriter_Name_OverrideChangesCameraAndStreamNames(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		SubStream:  baseSubRTSP,
		Name:       "porch",
	})
	assertContains(t, cfg, "  porch:\n")
	assertContains(t, cfg, "    'porch_main':\n")
	assertContains(t, cfg, "    'porch_sub':\n")
	assertContains(t, cfg, "- path: rtsp://127.0.0.1:8554/porch_main\n")
	assertContains(t, cfg, "- path: rtsp://127.0.0.1:8554/porch_sub\n")
}

// --- extractIP / buildInfo fallbacks ------------------------------------------

// URL without any parseable host -> camera/stream default names.
func TestWriter_BuildInfo_NoHostFallback(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: "rtsp:///nohost/stream"})
	assertContains(t, cfg, "  camera:\n")
	assertContains(t, cfg, "    'stream_main':\n")
}

// Hostname (non-IP) is used as-is without dot-sanitization via reIPv4.
func TestWriter_BuildInfo_HostnameUsed(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: "rtsp://cam.local:554/stream"})
	// hostname = "cam.local" -> sanitized "cam_local"
	assertContains(t, cfg, "  camera_cam_local:\n")
	assertContains(t, cfg, "    'cam_local_main':\n")
}

// --- Generate entry-point errors ----------------------------------------------

func TestGenerate_EmptyMainStreamErrors(t *testing.T) {
	_, err := Generate(&Request{})
	if err == nil {
		t.Fatal("expected error for empty MainStream")
	}
	if !strings.Contains(err.Error(), "mainStream required") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Response.Added: fresh config -> all lines are new (1..N).
func TestGenerate_Added_FreshConfigAllLines(t *testing.T) {
	resp, err := Generate(&Request{MainStream: baseRTSP})
	if err != nil {
		t.Fatal(err)
	}
	totalLines := strings.Count(resp.Config, "\n") + 1
	if len(resp.Added) != totalLines {
		t.Errorf("expected Added to cover all %d lines, got %d", totalLines, len(resp.Added))
	}
	// strictly increasing 1..N
	for i, n := range resp.Added {
		if n != i+1 {
			t.Errorf("Added[%d] = %d, want %d", i, n, i+1)
			break
		}
	}
}

// Response.Added: adding to existing config -> only new lines are flagged,
// and their indices (1-based) point to lines actually present in Config.
func TestGenerate_Added_IncrementalConfigOnlyNewLines(t *testing.T) {
	c1, err := Generate(&Request{MainStream: baseRTSP})
	if err != nil {
		t.Fatal(err)
	}
	c2, err := Generate(&Request{
		MainStream:     "rtsp://admin:pw@10.0.20.20:554/stream",
		ExistingConfig: c1.Config,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(c2.Added) == 0 {
		t.Fatal("expected some Added lines")
	}
	resultLines := strings.Split(c2.Config, "\n")
	for _, n := range c2.Added {
		if n < 1 || n > len(resultLines) {
			t.Errorf("Added line %d out of bounds (1..%d)", n, len(resultLines))
		}
	}
	// must be less than total (otherwise nothing was preserved)
	if len(c2.Added) >= len(resultLines) {
		t.Errorf("Added covers %d of %d lines -- expected partial", len(c2.Added), len(resultLines))
	}
}

// --- top-level structure stability --------------------------------------------

// Required top-level headers + order (mqtt -> record -> go2rtc -> cameras -> version).
func TestWriter_TopLevel_Order(t *testing.T) {
	cfg := mustGen(t, &Request{MainStream: baseRTSP})

	iMQTT := strings.Index(cfg, "mqtt:")
	iGlobalRec := strings.Index(cfg, "\nrecord:\n  enabled: true")
	iGo2rtc := strings.Index(cfg, "\ngo2rtc:")
	iCameras := strings.Index(cfg, "\ncameras:")
	iVersion := strings.Index(cfg, "\nversion:")

	if iMQTT < 0 || iGlobalRec < 0 || iGo2rtc < 0 || iCameras < 0 || iVersion < 0 {
		t.Fatalf("missing top-level section:\n%s", cfg)
	}
	if !(iMQTT < iGlobalRec && iGlobalRec < iGo2rtc && iGo2rtc < iCameras && iCameras < iVersion) {
		t.Errorf("wrong top-level order: mqtt=%d record=%d go2rtc=%d cameras=%d version=%d",
			iMQTT, iGlobalRec, iGo2rtc, iCameras, iVersion)
	}
}

// Section order inside a camera block (writer.go sequence).
func TestWriter_CameraBlock_SectionOrder(t *testing.T) {
	cfg := mustGen(t, &Request{
		MainStream: baseRTSP,
		SubStream:  baseSubRTSP,
		Live:       &LiveConfig{Height: 720},
		Detect:     &DetectConfig{Enabled: true},
		Objects:    []string{"person"},
		Motion:     &MotionConfig{Enabled: true},
		Record:     &RecordConfig{Enabled: true},
		Snapshots:  &BoolConfig{Enabled: true},
		Audio:      &AudioConfig{Enabled: true},
		Birdseye:   &BirdseyeConfig{Enabled: true},
		ONVIF:      &ONVIFConfig{Host: "10.0.20.10"},
		Notifications: &BoolConfig{Enabled: true},
		UI:            &UIConfig{Dashboard: false},
	})

	order := []string{
		"    ffmpeg:\n",
		"    live:\n",
		"    detect:\n",
		"    objects:\n",
		"    motion:\n",
		"    record:\n      enabled:",
		"    snapshots:\n",
		"    audio:\n",
		"    birdseye:\n",
		"    onvif:\n",
		"    notifications:\n",
		"    ui:\n",
	}
	prev := -1
	for _, s := range order {
		idx := strings.Index(cfg, s)
		if idx < 0 {
			t.Errorf("missing section %q\n%s", s, cfg)
			continue
		}
		if idx < prev {
			t.Errorf("section %q out of order\n%s", s, cfg)
		}
		prev = idx
	}
}
