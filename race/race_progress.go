package race

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"wwfc/logging"

	"github.com/logrusorgru/aurora/v3"
)

const (
	MaxDelays       = 20  // Maximum delays to track in rolling window
	SmoothingFactor = 0.3 // For exponential smoothing algorithm
)

type DelayMeasurement struct {
	Timestamp     int64   // Server timestamp when delay was measured
	RawDelay      float64 // Raw delay measurement
	SmoothedDelay float64 // Smoothed delay using exponential smoothing
}

type RaceProgressTiming struct {
	ClientStartTime int64              // Client's race start time (absolute timestamp)
	ServerStartTime int64              // Server's race start time (absolute timestamp)
	RecentDelays    []float64          // Rolling window of recent delays
	DelayData       []DelayMeasurement // All delay measurements for this race
}

// Global map to track race progress timing data by profile ID
var RaceProgressTimings = make(map[uint32]*RaceProgressTiming)

// logFinalRaceDelay logs the final smoothed and unsmoothed delays for a race to a separate file
func logFinalRaceDelay(pid uint32, finalSmoothedDelay float64, finalUnsmoothedDelay float64) {
	file, err := os.OpenFile("delay_logs/race_progress_delays.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logging.Error("race", "Failed to open race progress delays log file:", err.Error())
		return
	}
	defer file.Close()

	// Write to file: PID,Final_Smoothed_Delay,Final_Unsmoothed_Delay
	logEntry := fmt.Sprintf("%d,%.2f,%.2f\n", pid, finalSmoothedDelay, finalUnsmoothedDelay)
	_, err = file.WriteString(logEntry)
	if err != nil {
		logging.Error("race", "Failed to write to race progress delays log file:", err.Error())
	}
}

// logRaceProgressDelay stores delay measurements in memory for later batch writing
func logRaceProgressDelay(pid uint32, timestamp int64, rawDelay float64, smoothedDelay float64) {
	// Get the race start time to use as the race identifier
	if timing, exists := RaceProgressTimings[pid]; exists {
		// Store delay data in memory for batch writing at race end
		// We'll add a field to RaceProgressTiming to store the delay data
		if timing.DelayData == nil {
			timing.DelayData = make([]DelayMeasurement, 0)
		}

		measurement := DelayMeasurement{
			Timestamp:     timestamp,
			RawDelay:      rawDelay,
			SmoothedDelay: smoothedDelay,
		}
		timing.DelayData = append(timing.DelayData, measurement)
	}
}

// addDelayToWindow adds a new delay to the rolling window and returns the smoothed delay
func addDelayToWindow(timing *RaceProgressTiming, newDelay float64) float64 {
	// Add new delay to the window
	timing.RecentDelays = append(timing.RecentDelays, newDelay)

	// Maintain rolling window size
	if len(timing.RecentDelays) > MaxDelays {
		timing.RecentDelays = timing.RecentDelays[1:]
	}

	// Calculate smoothed delay using exponential smoothing
	var smoothedDelay float64
	if len(timing.RecentDelays) == 1 {
		// First delay, no smoothing needed
		smoothedDelay = newDelay
	} else {
		// Apply exponential smoothing
		previousSmoothed := timing.RecentDelays[len(timing.RecentDelays)-2]
		smoothedDelay = SmoothingFactor*newDelay + (1-SmoothingFactor)*previousSmoothed
	}

	return smoothedDelay
}

// HandleRaceProgressTime handles the wl:mkw_race_progress_time report case
func HandleRaceProgressTime(pid uint32, value string) {
	// Parse client absolute timestamp (this is the client's current time)
	clientTimestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		logging.Error("race", "Failed to parse client progress timestamp:", err.Error())
		return
	}

	serverTime := time.Now().UnixMilli()

	// logging.Info("race", "Race progress time:", aurora.Yellow(value), "Server time:", aurora.Yellow(strconv.FormatInt(serverTime, 10)))

	// Get or create progress timing for this player
	timing, exists := RaceProgressTimings[pid]
	if !exists {
		logging.Warn("race", "No race start timing found for progress report from profile", aurora.BrightCyan(strconv.FormatUint(uint64(pid), 10)))
		return
	}

	// Calculate client elapsed time: current client timestamp - client start timestamp
	clientElapsedTime := clientTimestamp - timing.ClientStartTime

	// Calculate server elapsed time: current server time - server start time
	serverElapsedTime := serverTime - timing.ServerStartTime

	// Calculate delay: client elapsed time - server elapsed time
	// This gives us the time difference between what the client thinks has passed vs what the server thinks has passed
	delay := float64(clientElapsedTime - serverElapsedTime)

	// Add to rolling window and get smoothed delay
	smoothedDelay := addDelayToWindow(timing, delay)

	// Log this delay measurement to per-race CSV file
	logRaceProgressDelay(pid, serverTime, delay, smoothedDelay)

	// logging.Info("race",
	// 	"Progress delay:", aurora.Cyan(fmt.Sprintf("%.2f", delay)),
	// 	"Smoothed delay:", aurora.Cyan(fmt.Sprintf("%.2f", smoothedDelay)),
	// 	"Delays tracked:", aurora.Cyan(strconv.Itoa(len(timing.RecentDelays))))

	// Progress delays are tracked in memory for smoothing calculation
	// Final delays are logged when race finishes
}

// LogRaceProgressDelay logs final race delays and cleans up timing data
func LogRaceProgressDelay(pid uint32, clientFinishTime int64, serverFinishTime int64) {
	if timing, exists := RaceProgressTimings[pid]; exists {
		// Calculate final unsmoothed delay using start and finish timestamps
		finalUnsmoothedDelay := float64((clientFinishTime - timing.ClientStartTime) - (serverFinishTime - timing.ServerStartTime))

		// Get final smoothed delay from the rolling window
		var finalSmoothedDelay float64
		if len(timing.RecentDelays) > 0 {
			finalSmoothedDelay = timing.RecentDelays[len(timing.RecentDelays)-1]
		}

		// Log both delays
		logFinalRaceDelay(pid, finalSmoothedDelay, finalUnsmoothedDelay)

		// Write all delay measurements to CSV file in batch
		writeRaceProgressDelaysToFile(pid, timing)

		// Clean up timing data for this player
		delete(RaceProgressTimings, pid)
	}
}

// writeRaceProgressDelaysToFile writes all stored delay measurements to a CSV file
func writeRaceProgressDelaysToFile(pid uint32, timing *RaceProgressTiming) {
	// Create filename with PID and race start time for uniqueness (one file per race per player)
	filename := fmt.Sprintf("delay_logs/race_delays_%d_%d.csv", pid, timing.ClientStartTime)

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logging.Error("race", "Failed to open race delay CSV file:", err.Error())
		return
	}
	defer file.Close()

	// Write header if file is empty
	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		header := "timestamp,raw_delay,smoothed_delay\n"
		_, err = file.WriteString(header)
		if err != nil {
			logging.Error("race", "Failed to write CSV header:", err.Error())
			return
		}
	}

	// Write all delay measurements in batch
	for _, measurement := range timing.DelayData {
		logEntry := fmt.Sprintf("%d,%.2f,%.2f\n", measurement.Timestamp, measurement.RawDelay, measurement.SmoothedDelay)
		_, err = file.WriteString(logEntry)
		if err != nil {
			logging.Error("race", "Failed to write to race delay CSV file:", err.Error())
			return
		}
	}
}

// CleanupRaceProgressTiming cleans up race progress timing data for a player
func CleanupRaceProgressTiming(pid uint32) {
	delete(RaceProgressTimings, pid)
}
