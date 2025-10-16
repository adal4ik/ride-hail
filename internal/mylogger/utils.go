package mylogger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const requestIDFile = "request_id.txt"

// generateDeterministicRequestID generates a deterministic request ID in the format 'startup-001', 'startup-002', etc.
func generateRequestID() (string, error) {
	// Read the last request ID from the file
	lastID, err := readLastRequestID()
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, start with 'startup-001'
			lastID = 0
		} else {
			return "", err
		}
	}

	// Increment the request ID
	nextID := lastID + 1

	// Format the request ID as 'startup-001', 'startup-002', etc.
	requestID := fmt.Sprintf("startup-%03d", nextID)

	// Save the updated request ID back to the file
	if err := writeLastRequestID(nextID); err != nil {
		return "", err
	}

	return requestID, nil
}

// readLastRequestID reads the last used request ID from the file.
func readLastRequestID() (int, error) {
	// Check if the file exists
	file, err := os.Open(requestIDFile)
	if err != nil {
		return 0, err // If the file doesn't exist, return 0 (i.e., starting from 'startup-001')
	}
	defer file.Close()

	var lastID int
	_, err = fmt.Fscanf(file, "%d", &lastID)
	if err != nil {
		return 0, err // If reading fails, return 0
	}

	return lastID, nil
}

// writeLastRequestID writes the updated request ID to the file.
func writeLastRequestID(lastID int) error {
	file, err := os.Create(requestIDFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Save the new request ID
	_, err = fmt.Fprintf(file, "%d", lastID)
	return err
}

// captureFrames collects stack trace frames
func captureFrames(skip, depth int) []stackFrame {
	pc := make([]uintptr, depth)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])

	var stack []stackFrame
	for {
		frame, more := frames.Next()
		stack = append(stack, stackFrame{
			Func:   filepath.Base(frame.Function),
			Source: filepath.Join(filepath.Base(filepath.Dir(frame.File)), filepath.Base(frame.File)),
			Line:   frame.Line,
		})
		if !more {
			break
		}
	}
	return stack
}

// stackFrame structure for capturing the stack trace
type stackFrame struct {
	Func   string `json:"func"`
	Source string `json:"source"`
	Line   int    `json:"line"`
}
