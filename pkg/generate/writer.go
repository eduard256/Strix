package generate

import (
	"fmt"
	"sort"
	"strings"
)

func writeStreamLines(b *strings.Builder, info *cameraInfo) {
	fmt.Fprintf(b, "    '%s':\n", info.MainStreamName)
	fmt.Fprintf(b, "      - %s\n", info.MainSource)

	if info.SubStreamName != "" {
		fmt.Fprintf(b, "    '%s':\n", info.SubStreamName)
		fmt.Fprintf(b, "      - %s\n", info.SubSource)
	}

	b.WriteByte('\n')
}

// writeCredentials writes top-level credential sections (xiaomi, tapo, ring, ...)
// populated by registered ExtractFunc handlers. Sorted by section, then by key.
// ex.
//   xiaomi:
//     "4161148305": V1:9d2w...
func writeCredentials(b *strings.Builder, creds map[string]map[string]string) {
	if len(creds) == 0 {
		return
	}

	sections := make([]string, 0, len(creds))
	for s := range creds {
		sections = append(sections, s)
	}
	sort.Strings(sections)

	for _, section := range sections {
		fmt.Fprintf(b, "%s:\n", section)

		keys := make([]string, 0, len(creds[section]))
		for k := range creds[section] {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Fprintf(b, "  %q: %s\n", k, creds[section][k])
		}
		b.WriteByte('\n')
	}
}

func writeCameraBlock(b *strings.Builder, info *cameraInfo, req *Request) {
	fmt.Fprintf(b, "  %s:\n", info.CameraName)

	b.WriteString("    ffmpeg:\n")
	writeFFmpegGlobal(b, req)
	b.WriteString("      inputs:\n")

	if info.SubStreamName != "" {
		writeInput(b, info.SubPath, info.SubInputArgs, "detect")
		writeInput(b, info.MainPath, info.MainInputArgs, "record")
	} else {
		writeInput(b, info.MainPath, info.MainInputArgs, "detect", "record")
	}

	writeLive(b, info, req)
	writeDetect(b, req)
	writeObjects(b, req)
	writeMotion(b, req)
	writeRecord(b, req)
	writeSnapshots(b, req)
	writeAudio(b, req)
	writeBirdseye(b, req)
	writeONVIF(b, req)
	writeNotifications(b, req)
	writeUI(b, req)

	b.WriteByte('\n')
}

func writeInput(b *strings.Builder, path, inputArgs string, roles ...string) {
	fmt.Fprintf(b, "        - path: %s\n", path)
	fmt.Fprintf(b, "          input_args: %s\n", inputArgs)
	b.WriteString("          roles:\n")
	for _, r := range roles {
		fmt.Fprintf(b, "            - %s\n", r)
	}
}

func writeFFmpegGlobal(b *strings.Builder, req *Request) {
	if req.FFmpeg == nil {
		return
	}
	if req.FFmpeg.HWAccel != "" && req.FFmpeg.HWAccel != "auto" {
		fmt.Fprintf(b, "      hwaccel_args: %s\n", req.FFmpeg.HWAccel)
	}
	if req.FFmpeg.GPU > 0 {
		fmt.Fprintf(b, "      gpu: %d\n", req.FFmpeg.GPU)
	}
}

func writeLive(b *strings.Builder, info *cameraInfo, req *Request) {
	if info.SubStreamName == "" && req.Live == nil {
		return
	}

	b.WriteString("    live:\n")

	if info.SubStreamName != "" {
		b.WriteString("      streams:\n")
		fmt.Fprintf(b, "        Main Stream: %s\n", info.MainStreamName)
		fmt.Fprintf(b, "        Sub Stream: %s\n", info.SubStreamName)
	}

	if req.Live != nil {
		if req.Live.Height > 0 {
			fmt.Fprintf(b, "      height: %d\n", req.Live.Height)
		}
		if req.Live.Quality > 0 {
			fmt.Fprintf(b, "      quality: %d\n", req.Live.Quality)
		}
	}
}

func writeDetect(b *strings.Builder, req *Request) {
	if req.Detect == nil {
		b.WriteString("    detect:\n      enabled: true\n")
		return
	}

	b.WriteString("    detect:\n")
	fmt.Fprintf(b, "      enabled: %t\n", req.Detect.Enabled)
	if req.Detect.FPS > 0 {
		fmt.Fprintf(b, "      fps: %d\n", req.Detect.FPS)
	}
	if req.Detect.Width > 0 {
		fmt.Fprintf(b, "      width: %d\n", req.Detect.Width)
	}
	if req.Detect.Height > 0 {
		fmt.Fprintf(b, "      height: %d\n", req.Detect.Height)
	}
}

func writeObjects(b *strings.Builder, req *Request) {
	objects := req.Objects
	if len(objects) == 0 {
		objects = []string{"person"}
	}

	b.WriteString("    objects:\n      track:\n")
	for _, obj := range objects {
		fmt.Fprintf(b, "        - %s\n", obj)
	}
}

func writeMotion(b *strings.Builder, req *Request) {
	if req.Motion == nil {
		return
	}

	b.WriteString("    motion:\n")
	fmt.Fprintf(b, "      enabled: %t\n", req.Motion.Enabled)
	if req.Motion.Threshold > 0 {
		fmt.Fprintf(b, "      threshold: %d\n", req.Motion.Threshold)
	}
	if req.Motion.ContourArea > 0 {
		fmt.Fprintf(b, "      contour_area: %d\n", req.Motion.ContourArea)
	}
}

func writeRecord(b *strings.Builder, req *Request) {
	if req.Record == nil {
		b.WriteString("    record:\n      enabled: true\n")
		return
	}

	b.WriteString("    record:\n")
	fmt.Fprintf(b, "      enabled: %t\n", req.Record.Enabled)

	if req.Record.RetainDays > 0 || req.Record.Mode != "" {
		b.WriteString("      retain:\n")
		if req.Record.RetainDays > 0 {
			fmt.Fprintf(b, "        days: %g\n", req.Record.RetainDays)
		}
		if req.Record.Mode != "" {
			fmt.Fprintf(b, "        mode: %s\n", req.Record.Mode)
		}
	}

	if req.Record.AlertsDays > 0 || req.Record.PreCapture > 0 || req.Record.PostCapture > 0 {
		b.WriteString("      alerts:\n")
		if req.Record.AlertsDays > 0 {
			fmt.Fprintf(b, "        retain:\n          days: %g\n", req.Record.AlertsDays)
		}
		if req.Record.PreCapture > 0 {
			fmt.Fprintf(b, "        pre_capture: %d\n", req.Record.PreCapture)
		}
		if req.Record.PostCapture > 0 {
			fmt.Fprintf(b, "        post_capture: %d\n", req.Record.PostCapture)
		}
	}

	if req.Record.DetectionDays > 0 {
		fmt.Fprintf(b, "      detections:\n        retain:\n          days: %g\n", req.Record.DetectionDays)
	}
}

func writeSnapshots(b *strings.Builder, req *Request) {
	if req.Snapshots == nil || !req.Snapshots.Enabled {
		return
	}
	b.WriteString("    snapshots:\n      enabled: true\n")
}

func writeAudio(b *strings.Builder, req *Request) {
	if req.Audio == nil || !req.Audio.Enabled {
		return
	}

	b.WriteString("    audio:\n      enabled: true\n")
	if len(req.Audio.Filters) > 0 {
		b.WriteString("      filters:\n")
		for _, f := range req.Audio.Filters {
			fmt.Fprintf(b, "        - %s\n", f)
		}
	}
}

func writeBirdseye(b *strings.Builder, req *Request) {
	if req.Birdseye == nil {
		return
	}

	b.WriteString("    birdseye:\n")
	fmt.Fprintf(b, "      enabled: %t\n", req.Birdseye.Enabled)
	if req.Birdseye.Mode != "" {
		fmt.Fprintf(b, "      mode: %s\n", req.Birdseye.Mode)
	}
}

func writeONVIF(b *strings.Builder, req *Request) {
	if req.ONVIF == nil || req.ONVIF.Host == "" {
		return
	}

	b.WriteString("    onvif:\n")
	fmt.Fprintf(b, "      host: %s\n", req.ONVIF.Host)

	port := req.ONVIF.Port
	if port == 0 {
		port = 80
	}
	fmt.Fprintf(b, "      port: %d\n", port)

	if req.ONVIF.User != "" {
		fmt.Fprintf(b, "      user: %s\n", req.ONVIF.User)
		fmt.Fprintf(b, "      password: %s\n", req.ONVIF.Password)
	}

	if req.ONVIF.AutoTracking {
		b.WriteString("      autotracking:\n        enabled: true\n")
		if len(req.ONVIF.RequiredZones) > 0 {
			b.WriteString("        required_zones:\n")
			for _, z := range req.ONVIF.RequiredZones {
				fmt.Fprintf(b, "          - %s\n", z)
			}
		}
	}

	if req.PTZ != nil && len(req.PTZ.Presets) > 0 {
		b.WriteString("      ptz:\n        presets:\n")
		for name, token := range req.PTZ.Presets {
			fmt.Fprintf(b, "          %s: %s\n", name, token)
		}
	}
}

func writeNotifications(b *strings.Builder, req *Request) {
	if req.Notifications == nil || !req.Notifications.Enabled {
		return
	}
	b.WriteString("    notifications:\n      enabled: true\n")
}

func writeUI(b *strings.Builder, req *Request) {
	if req.UI == nil {
		return
	}

	b.WriteString("    ui:\n")
	if req.UI.Order > 0 {
		fmt.Fprintf(b, "      order: %d\n", req.UI.Order)
	}
	if !req.UI.Dashboard {
		b.WriteString("      dashboard: false\n")
	}
}
