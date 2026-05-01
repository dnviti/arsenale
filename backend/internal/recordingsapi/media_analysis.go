package recordingsapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func (s Service) AnalyzeRecording(_ context.Context, item recordingResponse) (recordingAnalysisResponse, error) {
	if item.Format != "guac" {
		return recordingAnalysisResponse{}, &requestError{status: http.StatusBadRequest, message: "Only .guac recordings can be analyzed"}
	}

	file, err := os.Open(item.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return recordingAnalysisResponse{}, &requestError{status: http.StatusNotFound, message: "Recording file not found on disk"}
		}
		return recordingAnalysisResponse{}, fmt.Errorf("open recording: %w", err)
	}
	defer file.Close()

	payload, err := io.ReadAll(io.LimitReader(file, maxAnalyzeBytes+1))
	if err != nil {
		return recordingAnalysisResponse{}, fmt.Errorf("read recording: %w", err)
	}
	truncated := len(payload) > maxAnalyzeBytes
	if truncated {
		payload = payload[:maxAnalyzeBytes]
	}
	content := string(payload)

	instructions := make(map[string]int)
	displayWidth := 0
	displayHeight := 0
	hasLayer0Image := false

	for pos := 0; pos < len(content); {
		for pos < len(content) {
			switch content[pos] {
			case '\n', '\r', '\t', ' ':
				pos++
			default:
				goto parseInstruction
			}
		}
		break

	parseInstruction:
		semi := strings.IndexByte(content[pos:], ';')
		if semi == -1 {
			break
		}
		semi += pos
		raw := content[pos : semi+1]
		pos = semi + 1

		dot := strings.IndexByte(raw, '.')
		if dot == -1 {
			continue
		}
		opcodeLen, err := strconv.Atoi(raw[:dot])
		if err != nil || opcodeLen <= 0 || dot+1+opcodeLen > len(raw) {
			continue
		}
		opcode := raw[dot+1 : dot+1+opcodeLen]
		instructions[opcode]++

		switch opcode {
		case "size":
			parts := parseGuacArgs(raw)
			if len(parts) >= 3 && parts[0] == "0" {
				if value, err := strconv.Atoi(parts[1]); err == nil {
					displayWidth = value
				}
				if value, err := strconv.Atoi(parts[2]); err == nil {
					displayHeight = value
				}
			}
		case "img":
			if hasLayer0Image {
				continue
			}
			parts := parseGuacArgs(raw)
			if len(parts) >= 2 && parts[1] == "0" {
				hasLayer0Image = true
			}
		}
	}

	return recordingAnalysisResponse{
		FileSize:       len(payload),
		Truncated:      truncated,
		Instructions:   instructions,
		SyncCount:      instructions["sync"],
		DisplayWidth:   displayWidth,
		DisplayHeight:  displayHeight,
		HasLayer0Image: hasLayer0Image,
	}, nil
}

func parseGuacArgs(instruction string) []string {
	trimmed := strings.TrimSuffix(instruction, ";")
	args := make([]string, 0, 8)
	for pos := 0; pos < len(trimmed); {
		dot := strings.IndexByte(trimmed[pos:], '.')
		if dot == -1 {
			break
		}
		dot += pos
		length, err := strconv.Atoi(trimmed[pos:dot])
		if err != nil || length < 0 {
			break
		}
		start := dot + 1
		end := start + length
		if end > len(trimmed) {
			break
		}
		args = append(args, trimmed[start:end])
		pos = end
		if pos >= len(trimmed) {
			break
		}
		if trimmed[pos] != ',' {
			break
		}
		pos++
	}
	if len(args) <= 1 {
		return []string{}
	}
	return args[1:]
}
