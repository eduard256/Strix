package generate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	reCamerasHeader = regexp.MustCompile(`^cameras:`)
	reTopLevel      = regexp.MustCompile(`^[a-z]`)
	reCameraName    = regexp.MustCompile(`^\s{2}(\w[\w-]*):`)
	reStreamsHeader  = regexp.MustCompile(`^\s{2}streams:`)
	reStreamName    = regexp.MustCompile(`^\s{4}'?(\w[\w-]*)'?:`)
	reStreamContent = regexp.MustCompile(`^\s{4,}`)
	reNextSection   = regexp.MustCompile(`^[a-z#]`)
	reSibling       = regexp.MustCompile(`^  \w`) // sibling under go2rtc: (ex. `  xiaomi:`)
	reCameraBody    = regexp.MustCompile(`^\s{2,}\S`)
	reVersion       = regexp.MustCompile(`^version:`)
)

func addToConfig(existing string, info *cameraInfo, req *Request) (*Response, error) {
	lines := strings.Split(existing, "\n")

	existingCams := findNames(lines, reCamerasHeader, reCameraName)
	existingStreams := findNames(lines, reStreamsHeader, reStreamName)

	info = dedup(info, existingCams, existingStreams)

	streamIdx := findStreamInsertPoint(lines)
	cameraIdx := findCameraInsertPoint(lines)

	if streamIdx == -1 || cameraIdx == -1 {
		return nil, fmt.Errorf("generate: can't find go2rtc streams or cameras section")
	}

	var sb strings.Builder
	writeStreamLines(&sb, info)
	streamLines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")

	sb.Reset()
	writeCameraBlock(&sb, info, req)
	cameraLines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")

	added := make(map[int]bool)
	result := make([]string, 0, len(lines)+len(streamLines)+len(cameraLines))

	result = append(result, lines[:streamIdx]...)
	mark := len(result)
	result = append(result, streamLines...)
	for i := range streamLines {
		added[mark+i] = true
	}

	shift := len(streamLines)
	adjCameraIdx := cameraIdx + shift
	rest := lines[streamIdx:]
	split := adjCameraIdx - len(result)

	result = append(result, rest[:split]...)
	mark = len(result)
	result = append(result, cameraLines...)
	for i := range cameraLines {
		added[mark+i] = true
	}
	result = append(result, rest[split:]...)

	// upsert credential sections (xiaomi, tapo, ...) before cameras:
	result, added = upsertCredentials(result, info.Credentials, added)

	config := strings.Join(result, "\n")

	addedLines := make([]int, 0, len(added))
	for i := range result {
		if added[i] {
			addedLines = append(addedLines, i+1)
		}
	}

	return &Response{Config: config, Added: addedLines}, nil
}

func dedup(info *cameraInfo, cams, streams map[string]bool) *cameraInfo {
	out := *info

	suffix := 0
	base := out.CameraName
	for cams[out.CameraName] {
		suffix++
		out.CameraName = fmt.Sprintf("%s_%d", base, suffix)
	}

	base = out.MainStreamName
	for streams[out.MainStreamName] {
		suffix++
		out.MainStreamName = fmt.Sprintf("%s_%d", base, suffix)
	}

	if out.SubStreamName != "" {
		base = out.SubStreamName
		for streams[out.SubStreamName] {
			suffix++
			out.SubStreamName = fmt.Sprintf("%s_%d", base, suffix)
		}
	}

	return &out
}

func findNames(lines []string, header, nameRe *regexp.Regexp) map[string]bool {
	names := make(map[string]bool)
	in := false
	for _, line := range lines {
		if header.MatchString(line) {
			in = true
			continue
		}
		if in && reTopLevel.MatchString(line) {
			break
		}
		if in {
			if m := nameRe.FindStringSubmatch(line); m != nil {
				names[m[1]] = true
			}
		}
	}
	return names
}

func findStreamInsertPoint(lines []string) int {
	in := false
	last := -1
	headerIdx := -1
	for i, line := range lines {
		if reStreamsHeader.MatchString(line) {
			in = true
			headerIdx = i
			continue
		}
		if in {
			// stop at sibling under go2rtc: (ex. `  xiaomi:`)
			if reSibling.MatchString(line) && !reStreamsHeader.MatchString(line) {
				if last >= 0 {
					return last + 1
				}
				return headerIdx + 1
			}
			if reStreamContent.MatchString(line) {
				last = i
			} else if reNextSection.MatchString(line) {
				if last >= 0 && last+1 < len(lines) && strings.TrimSpace(lines[last+1]) == "" {
					return last + 2
				}
				if last >= 0 {
					return last + 1
				}
				return headerIdx + 1
			}
		}
	}
	if last >= 0 {
		return last + 1
	}
	if headerIdx >= 0 {
		return headerIdx + 1
	}
	return -1
}

// upsertCredentials merges creds into credential sections nested under go2rtc:.
// For each section: if a matching line `    "<key>":` exists -- replace its
// value; else insert in sorted order. If the section itself doesn't exist --
// create a new nested block inside go2rtc: (after streams: block).
func upsertCredentials(lines []string, creds map[string]map[string]string, added map[int]bool) ([]string, map[int]bool) {
	if len(creds) == 0 {
		return lines, added
	}

	sections := make([]string, 0, len(creds))
	for s := range creds {
		sections = append(sections, s)
	}
	sort.Strings(sections)

	for _, section := range sections {
		lines, added = upsertSection(lines, section, creds[section], added)
	}

	return lines, added
}

// ex. `    "4161148305": V1:xxx` -- 4-space indent under nested section
var reCredKey = regexp.MustCompile(`^\s{4}"([^"]+)":`)

func upsertSection(lines []string, section string, kv map[string]string, added map[int]bool) ([]string, map[int]bool) {
	// section header is nested under go2rtc:, ex. `  xiaomi:`
	reHeader := regexp.MustCompile(`^  ` + regexp.QuoteMeta(section) + `:\s*$`)

	headerIdx := -1
	for i, line := range lines {
		if reHeader.MatchString(line) {
			headerIdx = i
			break
		}
	}

	if headerIdx == -1 {
		return insertNewSection(lines, section, kv, added)
	}

	// section exists -- find end (blank line, top-level header, or sibling 2-space key)
	end := len(lines)
	for i := headerIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" || reTopLevel.MatchString(line) {
			end = i
			break
		}
		// sibling under go2rtc: has 2-space indent, not 4
		if len(line) >= 2 && line[0] == ' ' && line[1] == ' ' && (len(line) == 2 || line[2] != ' ') {
			end = i
			break
		}
	}

	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		newLine := fmt.Sprintf("    %q: %s", k, kv[k])

		// try replace -- no length change, just mark modified line
		replaced := false
		for i := headerIdx + 1; i < end; i++ {
			if m := reCredKey.FindStringSubmatch(lines[i]); m != nil && m[1] == k {
				if lines[i] != newLine {
					lines[i] = newLine
					added[i] = true
				}
				replaced = true
				break
			}
		}
		if replaced {
			continue
		}

		// insert in sorted order within section
		insertAt := headerIdx + 1
		for i := headerIdx + 1; i < end; i++ {
			if m := reCredKey.FindStringSubmatch(lines[i]); m != nil {
				if m[1] < k {
					insertAt = i + 1
				} else {
					break
				}
			} else {
				insertAt = i + 1
			}
		}

		lines = append(lines[:insertAt], append([]string{newLine}, lines[insertAt:]...)...)
		added = shiftAdded(added, insertAt)
		added[insertAt] = true
		end++
	}

	return lines, added
}

// insertNewSection adds a new nested section under go2rtc:, after the streams:
// block but before any sibling go2rtc key or top-level header.
func insertNewSection(lines []string, section string, kv map[string]string, added map[int]bool) ([]string, map[int]bool) {
	// find end of streams: block inside go2rtc:
	insertAt := findGo2RTCInsertPoint(lines)
	if insertAt < 0 {
		return lines, added
	}

	block := []string{"  " + section + ":"}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		block = append(block, fmt.Sprintf("    %q: %s", k, kv[k]))
	}

	lines = append(lines[:insertAt], append(block, lines[insertAt:]...)...)
	added = shiftAdded(added, insertAt)
	for i := range block {
		added[insertAt+i] = true
	}

	return lines, added
}

// findGo2RTCInsertPoint returns the line index where a new nested section
// under go2rtc: should be inserted -- after the last non-blank content line
// of the go2rtc: block.
func findGo2RTCInsertPoint(lines []string) int {
	reGo2RTCHeader := regexp.MustCompile(`^go2rtc:\s*$`)

	headerIdx := -1
	for i, line := range lines {
		if reGo2RTCHeader.MatchString(line) {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		return -1
	}

	last := headerIdx
	for i := headerIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if reTopLevel.MatchString(line) {
			break
		}
		last = i
	}

	return last + 1
}

// shiftAdded moves all marks at index >= from by +1. Also used with from=len(lines)
// as a no-op shift (just return same map).
func shiftAdded(added map[int]bool, from int) map[int]bool {
	out := make(map[int]bool, len(added))
	for i := range added {
		if i >= from {
			out[i+1] = true
		} else {
			out[i] = true
		}
	}
	return out
}

func findCameraInsertPoint(lines []string) int {
	in := false
	last := -1
	headerIdx := -1
	for i, line := range lines {
		if reCamerasHeader.MatchString(line) {
			in = true
			headerIdx = i
			continue
		}
		if in {
			if reCameraBody.MatchString(line) {
				last = i
			} else if reTopLevel.MatchString(line) && !reCamerasHeader.MatchString(line) {
				if last < 0 {
					return headerIdx + 1
				}
				idx := last + 1
				for idx < len(lines) && strings.TrimSpace(lines[idx]) == "" {
					idx++
				}
				return idx
			} else if reVersion.MatchString(line) {
				if last < 0 {
					return headerIdx + 1
				}
				idx := i
				for idx > 0 && strings.TrimSpace(lines[idx-1]) == "" {
					idx--
				}
				return idx
			}
		}
	}
	if headerIdx >= 0 {
		return headerIdx + 1
	}
	return len(lines)
}
