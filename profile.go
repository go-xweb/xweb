package xweb
import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	runtimePprof "runtime/pprof"
	"strconv"
	"time"
)

var startTime = time.Now()
var pid int

func init() {
	pid = os.Getpid()
}

// start cpu profile monitor
func StartCPUProfile() {
	f, err := os.Create("cpu-" + strconv.Itoa(pid) + ".pprof")
	if err != nil {
		log.Fatal(err)
	}
	runtimePprof.StartCPUProfile(f)
}

// stop cpu profile monitor
func StopCPUProfile() {
	runtimePprof.StopCPUProfile()
}

// print gc information to io.Writer
func PrintGCSummary(w io.Writer) {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	gcstats := &debug.GCStats{PauseQuantiles: make([]time.Duration, 100)}
	debug.ReadGCStats(gcstats)

	printGC(memStats, gcstats, w)
}

func printGC(memStats *runtime.MemStats, gcstats *debug.GCStats, w io.Writer) {

	if gcstats.NumGC > 0 {
		lastPause := gcstats.Pause[0]
		elapsed := time.Now().Sub(startTime)
		overhead := float64(gcstats.PauseTotal) / float64(elapsed) * 100
		allocatedRate := float64(memStats.TotalAlloc) / elapsed.Seconds()

		fmt.Fprintf(w, "NumGC:%d Pause:%s Pause(Avg):%s Overhead:%3.2f%% Alloc:%s Sys:%s Alloc(Rate):%s/s Histogram:%s %s %s \n",
			gcstats.NumGC,
			FriendlyTime(lastPause),
			FriendlyTime(AvgTime(gcstats.Pause)),
			overhead,
			FriendlyBytes(memStats.Alloc),
			FriendlyBytes(memStats.Sys),
			FriendlyBytes(uint64(allocatedRate)),
			FriendlyTime(gcstats.PauseQuantiles[94]),
			FriendlyTime(gcstats.PauseQuantiles[98]),
			FriendlyTime(gcstats.PauseQuantiles[99]))
	} else {
		// while GC has disabled
		elapsed := time.Now().Sub(startTime)
		allocatedRate := float64(memStats.TotalAlloc) / elapsed.Seconds()

		fmt.Fprintf(w, "Alloc:%s Sys:%s Alloc(Rate):%s/s\n",
			FriendlyBytes(memStats.Alloc),
			FriendlyBytes(memStats.Sys),
			FriendlyBytes(uint64(allocatedRate)))
	}
}

func AvgTime(items []time.Duration) time.Duration {
	var sum time.Duration
	for _, item := range items {
		sum += item
	}
	return time.Duration(int64(sum) / int64(len(items)))
}

// format bytes number friendly
func FriendlyBytes(bytes uint64) string {
	units := [...]string{"YB", "ZB", "EB", "PB", "TB", "GB", "MB", "KB", "B"}
	total := len(units)
	for total--; total > 0 && bytes > 1024; total-- {
		bytes /= 1024
	}
	return fmt.Sprintf("%d%s", bytes, units[total])
}

// short string format
func FriendlyTime(d time.Duration) string {

	u := uint64(d)
	if u < uint64(time.Second) {
		switch {
		case u == 0:
			return "0"
		case u < uint64(time.Microsecond):
			return fmt.Sprintf("%.2fns", float64(u))
		case u < uint64(time.Millisecond):
			return fmt.Sprintf("%.2fus", float64(u)/1000)
		default:
			return fmt.Sprintf("%.2fms", float64(u)/1000/1000)
		}
	} else {
		switch {
		case u < uint64(time.Minute):
			return fmt.Sprintf("%.2fs", float64(u)/1000/1000/1000)
		case u < uint64(time.Hour):
			return fmt.Sprintf("%.2fm", float64(u)/1000/1000/1000/60)
		default:
			return fmt.Sprintf("%.2fh", float64(u)/1000/1000/1000/60/60)
		}
	}

}
