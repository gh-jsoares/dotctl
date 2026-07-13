package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold    = lipgloss.NewStyle().Bold(true)
	section = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
	msg    string
	stop   chan struct{}
	done   chan struct{}
	mu     sync.Mutex
	w      io.Writer
}

func NewSpinner(w io.Writer, msg string) *Spinner {
	return &Spinner{
		msg:  msg,
		stop: make(chan struct{}),
		done: make(chan struct{}),
		w:    w,
	}
}

func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				fmt.Fprintf(s.w, "\r\033[K")
				return
			case <-ticker.C:
				s.mu.Lock()
				frame := cyan.Render(spinnerFrames[i%len(spinnerFrames)])
				fmt.Fprintf(s.w, "\r\033[K  %s %s", frame, s.msg)
				s.mu.Unlock()
				i++
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
}

func Section(title string) {
	fmt.Fprintf(os.Stdout, "\n %s\n", section.Render(title))
}

func StepStart(name string) *StepTimer {
	return &StepTimer{
		name:  name,
		start: time.Now(),
	}
}

type StepTimer struct {
	name  string
	start time.Time
}

func (s *StepTimer) Success() {
	elapsed := time.Since(s.start)
	icon := green.Render("✓")
	timing := dim.Render(formatDuration(elapsed))
	fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, s.name, timing)
}

func (s *StepTimer) Skip(reason string) {
	icon := yellow.Render("○")
	msg := dim.Render(reason)
	fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, dim.Render(s.name), msg)
}

func (s *StepTimer) Fail(err error) {
	elapsed := time.Since(s.start)
	icon := red.Render("✗")
	timing := dim.Render(formatDuration(elapsed))
	fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, s.name, timing)
	fmt.Fprintf(os.Stderr, "    %s\n", red.Render(err.Error()))
}

func (s *StepTimer) Warn(err error) {
	elapsed := time.Since(s.start)
	icon := yellow.Render("⚠")
	timing := dim.Render(formatDuration(elapsed))
	fmt.Fprintf(os.Stdout, "  %s %s %s\n", icon, s.name, timing)
	fmt.Fprintf(os.Stderr, "    %s\n", yellow.Render(err.Error()))
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
