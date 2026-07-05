package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Section defines the interface for a renderable dashboard panel.
// Each section occupies a rectangular region and can be updated with new data.
type Section interface {
	// Render writes the section content to w within the given width and height bounds.
	Render(w io.Writer, width, height int)
	// Update provides new data to the section. The concrete type of data depends
	// on the section implementation.
	Update(data any)
}

// Dashboard renders a live TUI with panels for metrics, logs, and status.
// It uses the terminal's alternate screen buffer to avoid polluting scrollback
// and positions content using ANSI cursor control sequences.
type Dashboard struct {
	enabled  bool
	refresh  time.Duration
	cancel   context.CancelFunc
	output   io.Writer
	width    int
	height   int
	sections []Section
	mu       sync.Mutex
}

// NewDashboard creates a Dashboard configured with a metrics panel and a log panel.
// The log panel reads recent entries from the provided ring buffer.
func NewDashboard(output io.Writer, refresh time.Duration, ringBuf *RingBuffer[LogEntry]) *Dashboard {
	if output == nil {
		output = os.Stdout
	}
	if refresh <= 0 {
		refresh = defaultRefreshInterval
	}

	metrics := &MetricsPanel{}
	logPanel := &LogPanel{ringBuf: ringBuf}

	return &Dashboard{
		enabled:  true,
		refresh:  refresh,
		output:   output,
		width:    80,
		height:   24,
		sections: []Section{metrics, logPanel},
	}
}

// Start enters the alternate screen buffer, hides the cursor, and detects
// terminal dimensions. It performs one initial render and then enters the
// render loop which blocks until context cancellation.
func (d *Dashboard) Start(ctx context.Context) error {
	d.mu.Lock()
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.mu.Unlock()

	// Detect terminal size
	w, h := getTerminalSize()
	d.mu.Lock()
	d.width = w
	d.height = h
	d.mu.Unlock()

	// Enter alternate screen buffer and hide cursor
	enterAltScreen(d.output)
	hideCursor(d.output)
	clearScreen(d.output)

	// Render one initial frame
	d.render()

	// Enter render loop — blocks until context is cancelled
	d.renderLoop(ctx)

	// Restore terminal on exit
	d.Stop()
	return ctx.Err()
}

// renderLoop ticks at the configured refresh interval and re-renders the
// dashboard on each tick. It exits cleanly when ctx is cancelled.
func (d *Dashboard) renderLoop(ctx context.Context) {
	ticker := time.NewTicker(d.refresh)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.render()
		case <-ctx.Done():
			return
		}
	}
}

// Stop restores the terminal to its normal state: shows the cursor,
// exits the alternate screen buffer, and cancels the render loop.
func (d *Dashboard) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	showCursor(d.output)
	exitAltScreen(d.output)
}

// render performs a single frame render: clears the screen, positions the cursor
// at the origin, and renders each section sequentially.
func (d *Dashboard) render() {
	d.mu.Lock()
	defer d.mu.Unlock()

	clearScreen(d.output)
	moveCursor(d.output, 1, 1)

	if len(d.sections) == 0 {
		return
	}

	// Layout: MetricsPanel gets the top portion (fixed 5 lines),
	// LogPanel gets the remainder.
	metricsHeight := 5
	logHeight := d.height - metricsHeight - 1 // -1 for separator line

	if logHeight < 3 {
		logHeight = 3
	}

	// Render metrics panel at the top
	if len(d.sections) > 0 {
		moveCursor(d.output, 1, 1)
		d.sections[0].Render(d.output, d.width, metricsHeight)
	}

	// Render separator line
	separatorRow := metricsHeight + 1
	moveCursor(d.output, separatorRow, 1)
	for i := 0; i < d.width; i++ {
		fmt.Fprint(d.output, "─")
	}

	// Render log panel below separator
	if len(d.sections) > 1 {
		moveCursor(d.output, separatorRow+1, 1)
		d.sections[1].Render(d.output, d.width, logHeight)
	}
}

// Sections returns the dashboard's current sections.
func (d *Dashboard) Sections() []Section {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sections
}

// --- Terminal control functions ---

// enterAltScreen writes the escape sequence to enter the alternate screen buffer.
func enterAltScreen(w io.Writer) {
	fmt.Fprint(w, "\033[?1049h")
}

// exitAltScreen writes the escape sequence to exit the alternate screen buffer.
func exitAltScreen(w io.Writer) {
	fmt.Fprint(w, "\033[?1049l")
}

// moveCursor positions the terminal cursor at the given row and column (1-based).
func moveCursor(w io.Writer, row, col int) {
	fmt.Fprintf(w, "\033[%d;%dH", row, col)
}

// clearScreen writes the escape sequence to clear the entire screen.
func clearScreen(w io.Writer) {
	fmt.Fprint(w, "\033[2J")
}

// hideCursor writes the escape sequence to hide the terminal cursor.
func hideCursor(w io.Writer) {
	fmt.Fprint(w, "\033[?25l")
}

// showCursor writes the escape sequence to show the terminal cursor.
func showCursor(w io.Writer) {
	fmt.Fprint(w, "\033[?25h")
}

// getTerminalSize attempts to determine the terminal dimensions.
// Falls back to 80x24 if detection fails.
func getTerminalSize() (width, height int) {
	// Try to get terminal size from stdout
	w, h, err := terminalSize(os.Stdout.Fd())
	if err == nil && w > 0 && h > 0 {
		return w, h
	}
	// Fallback to standard terminal size
	return 80, 24
}
