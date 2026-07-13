package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	green       = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	red         = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cyan        = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dim         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold        = lipgloss.NewStyle().Bold(true)
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func Section(title string) {
	fmt.Fprintf(os.Stdout, "\n %s\n", sectionStyle.Render(title))
}

func SectionWithCount(title string, count int) {
	fmt.Fprintf(os.Stdout, "\n %s %s\n", sectionStyle.Render(title), dim.Render(fmt.Sprintf("(%d)", count)))
}

// LiveStep manages a running step with spinner and dimmed output.
type LiveStep struct {
	name    string
	counter string
	start   time.Time
	mu      sync.Mutex
	stop    chan struct{}
	done    chan struct{}
	frame   int
	active  bool
}

func StepStart(name string) *LiveStep {
	return &LiveStep{
		name:  name,
		start: time.Now(),
	}
}

func StepStartWithCounter(name string, current, total int) *LiveStep {
	return &LiveStep{
		name:    name,
		counter: dim.Render(fmt.Sprintf("[%d/%d]", current, total)),
		start:   time.Now(),
	}
}

// StartSpin begins the spinner animation. Call this when a step will take time.
func (s *LiveStep) StartSpin() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		defer close(s.done)
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				s.clearLine()
				return
			case <-ticker.C:
				s.mu.Lock()
				frame := cyan.Render(spinnerFrames[s.frame%len(spinnerFrames)])
				s.frame++
				s.mu.Unlock()
				s.clearLine()
				if s.counter != "" {
					fmt.Fprintf(os.Stdout, "\r  %s %s %s", frame, s.counter, s.name)
				} else {
					fmt.Fprintf(os.Stdout, "\r  %s %s", frame, s.name)
				}
			}
		}
	}()
}

// Writer returns an io.Writer that prints dimmed, indented output while
// coordinating with the spinner (clears spinner line, prints output, redraws).
func (s *LiveStep) Writer() io.Writer {
	return &stepWriter{step: s}
}

func (s *LiveStep) clearLine() {
	fmt.Fprintf(os.Stdout, "\r\033[K")
}

func (s *LiveStep) printDimLine(line string) {
	s.mu.Lock()
	wasActive := s.active
	s.mu.Unlock()

	if wasActive {
		s.clearLine()
	}
	fmt.Fprintf(os.Stdout, "    %s\n", dim.Render(line))
	if wasActive {
		s.mu.Lock()
		frame := cyan.Render(spinnerFrames[s.frame%len(spinnerFrames)])
		s.mu.Unlock()
		if s.counter != "" {
			fmt.Fprintf(os.Stdout, "\r  %s %s %s", frame, s.counter, s.name)
		} else {
			fmt.Fprintf(os.Stdout, "\r  %s %s", frame, s.name)
		}
	}
}

func (s *LiveStep) stopSpin() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()
	close(s.stop)
	<-s.done
}

func (s *LiveStep) Success() {
	s.stopSpin()
	elapsed := time.Since(s.start)
	icon := green.Render("✓")
	timing := dim.Render(formatDuration(elapsed))
	if s.counter != "" {
		fmt.Fprintf(os.Stdout, "  %s %s %s %s\n", icon, s.counter, s.name, timing)
	} else {
		fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, s.name, timing)
	}
}

func (s *LiveStep) Skip(reason string) {
	s.stopSpin()
	icon := yellow.Render("○")
	msg := dim.Render(reason)
	if s.counter != "" {
		fmt.Fprintf(os.Stdout, "  %s %s %s %s\n", icon, s.counter, dim.Render(s.name), msg)
	} else {
		fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, dim.Render(s.name), msg)
	}
}

func (s *LiveStep) Fail(err error) {
	s.stopSpin()
	elapsed := time.Since(s.start)
	icon := red.Render("✗")
	timing := dim.Render(formatDuration(elapsed))
	if s.counter != "" {
		fmt.Fprintf(os.Stdout, "  %s %s %s %s\n", icon, s.counter, s.name, timing)
	} else {
		fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, s.name, timing)
	}
	fmt.Fprintf(os.Stdout, "    %s\n", red.Render(err.Error()))
}

func (s *LiveStep) Warn(err error) {
	s.stopSpin()
	elapsed := time.Since(s.start)
	icon := yellow.Render("⚠")
	timing := dim.Render(formatDuration(elapsed))
	if s.counter != "" {
		fmt.Fprintf(os.Stdout, "  %s %s %s %s\n", icon, s.counter, s.name, timing)
	} else {
		fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, s.name, timing)
	}
	fmt.Fprintf(os.Stdout, "    %s\n", yellow.Render(err.Error()))
}

func Summary(total int, failed int, elapsed time.Duration) {
	fmt.Fprintln(os.Stdout)
	if failed == 0 {
		icon := green.Render("✓")
		fmt.Fprintf(os.Stdout, " %s %s %s\n",
			icon,
			bold.Render("Sync complete"),
			dim.Render(formatDuration(elapsed)),
		)
	} else {
		icon := red.Render("✗")
		fmt.Fprintf(os.Stdout, " %s %s %s\n",
			icon,
			bold.Render(fmt.Sprintf("Sync failed (%d/%d steps failed)", failed, total)),
			dim.Render(formatDuration(elapsed)),
		)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("(%dms)", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("(%.1fs)", d.Seconds())
	}
	return fmt.Sprintf("(%dm%ds)", int(d.Minutes()), int(d.Seconds())%60)
}

// stepWriter writes lines through the LiveStep's dimmed output channel.
type stepWriter struct {
	step *LiveStep
	buf  []byte
}

func (w *stepWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	for {
		idx := -1
		for i, b := range w.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:idx]), "\r")
		w.buf = w.buf[idx+1:]
		if line != "" {
			w.step.printDimLine(line)
		}
	}
	return len(p), nil
}

// Flush writes any remaining buffered content.
func (w *stepWriter) Flush() {
	if len(w.buf) > 0 {
		line := strings.TrimRight(string(w.buf), "\r\n")
		if line != "" {
			w.step.printDimLine(line)
		}
		w.buf = nil
	}
}

// PipeCmd is a helper to create a writer pair for use with exec.Cmd.
// Returns combined stdout+stderr writer. Call Flush() after cmd.Run().
type PipeCmd struct {
	w *stepWriter
}

func NewPipeCmd(step *LiveStep) *PipeCmd {
	return &PipeCmd{w: &stepWriter{step: step}}
}

func (p *PipeCmd) Writer() io.Writer {
	return p.w
}

func (p *PipeCmd) Flush() {
	p.w.Flush()
}

// ScanWriter creates an io.Writer that scans lines and calls a function for each.
// Useful for filtering or transforming output.
func ScanWriter(fn func(line string)) io.Writer {
	pr, pw := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			fn(scanner.Text())
		}
	}()
	return pw
}
