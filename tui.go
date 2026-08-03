package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chelnak/ysmrr/pkg/animations"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

const (
	defaultTerminalWidth = 80
	minFrameDuration     = 120 * time.Millisecond
	ansiBrightGreen      = "\x1b[92m"
	ansiBrightRed        = "\x1b[91m"
	ansiReset            = "\x1b[0m"
)

type taskRenderer interface {
	NewTask(message string) taskHandle
	Println(message string)
	BlankLine()
	Stop()
}

type taskHandle interface {
	Update(message string)
	Complete()
	Error()
}

var (
	_ taskRenderer = (*interactiveRenderer)(nil)
	_ taskRenderer = (*plainRenderer)(nil)
	_ taskRenderer = silentRenderer{}
	_ taskHandle   = (*interactiveTask)(nil)
	_ taskHandle   = (*plainTask)(nil)
	_ taskHandle   = silentTask{}
)

func newTaskRenderer(jsonOutput, unixOutput bool) taskRenderer {
	if jsonOutput && !unixOutput {
		return silentRenderer{}
	}
	if unixOutput || !stdoutIsTerminal() {
		return newPlainRenderer(os.Stdout)
	}

	frameDuration, frames := animations.GetAnimation(animations.Dots)
	if frameDuration < minFrameDuration {
		frameDuration = minFrameDuration
	}
	return newInteractiveRenderer(defaultTerminalWriter(), frames, frameDuration, terminalWidth)
}

func stdoutIsTerminal() bool {
	fd := os.Stdout.Fd()
	return os.Getenv("YSMRR_FORCE_TTY") == "true" ||
		isatty.IsTerminal(fd) ||
		isatty.IsCygwinTerminal(fd)
}

func defaultTerminalWriter() io.Writer {
	if runtime.GOOS == "windows" {
		return colorable.NewColorableStdout()
	}
	return os.Stdout
}

func terminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return defaultTerminalWidth
	}
	return width
}

type taskStatus uint8

const (
	taskRunning taskStatus = iota
	taskComplete
	taskError
)

type interactiveTask struct {
	renderer *interactiveRenderer
	message  string
	status   taskStatus
	raw      bool
}

type interactiveRenderer struct {
	mu            sync.Mutex
	writer        io.Writer
	tasks         []*interactiveTask
	frames        []string
	frameDuration time.Duration
	width         func() int
	stop          chan struct{}
	done          chan struct{}
	stopOnce      sync.Once
	renderedLines int
	frame         int
}

func newInteractiveRenderer(
	writer io.Writer,
	frames []string,
	frameDuration time.Duration,
	width func() int,
) *interactiveRenderer {
	if len(frames) == 0 {
		frames = []string{"-"}
	}
	if frameDuration <= 0 {
		frameDuration = minFrameDuration
	}
	if width == nil {
		width = func() int { return defaultTerminalWidth }
	}

	renderer := &interactiveRenderer{
		writer:        writer,
		frames:        frames,
		frameDuration: frameDuration,
		width:         width,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go renderer.run()
	return renderer
}

func (r *interactiveRenderer) NewTask(message string) taskHandle {
	task := &interactiveTask{renderer: r, message: message, status: taskRunning}
	r.mu.Lock()
	r.tasks = append(r.tasks, task)
	r.mu.Unlock()
	return task
}

func (r *interactiveRenderer) Println(message string) {
	task := &interactiveTask{renderer: r, message: message, status: taskComplete}
	r.mu.Lock()
	r.tasks = append(r.tasks, task)
	r.mu.Unlock()
}

func (r *interactiveRenderer) BlankLine() {
	task := &interactiveTask{renderer: r, status: taskComplete, raw: true}
	r.mu.Lock()
	r.tasks = append(r.tasks, task)
	r.mu.Unlock()
}

func (r *interactiveRenderer) Stop() {
	r.stopOnce.Do(func() {
		close(r.stop)
		<-r.done
	})
}

func (r *interactiveRenderer) run() {
	ticker := time.NewTicker(r.frameDuration)
	defer ticker.Stop()
	defer close(r.done)

	_, _ = fmt.Fprint(r.writer, "\x1b[?25l")
	defer func() {
		r.clearRenderedLines()
		_, _ = fmt.Fprint(r.writer, "\x1b[?25h")
	}()

	for {
		select {
		case <-ticker.C:
			r.render(false)
		case <-r.stop:
			r.render(true)
			return
		}
	}
}

func (r *interactiveRenderer) render(final bool) {
	finalLines, activeLines := r.collectLines(final)
	r.clearRenderedLines()

	for _, line := range finalLines {
		_, _ = fmt.Fprintln(r.writer, line)
	}
	for _, line := range activeLines {
		_, _ = fmt.Fprintln(r.writer, line)
	}
	r.renderedLines = len(activeLines)
}

func (r *interactiveRenderer) collectLines(final bool) ([]string, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	frame := r.frames[r.frame]
	r.frame = (r.frame + 1) % len(r.frames)
	width := r.width()
	finalLines := make([]string, 0)
	activeLines := make([]string, 0, len(r.tasks))
	activeTasks := make([]*interactiveTask, 0, len(r.tasks))

	for _, task := range r.tasks {
		if final || task.status != taskRunning {
			if task.raw {
				finalLines = append(finalLines, "")
			} else {
				line := formatTaskLine(task.message, task.status, frame, width)
				finalLines = append(finalLines, colorizeTaskLine(line, task.status))
			}
			continue
		}
		line := formatTaskLine(task.message, task.status, frame, width)
		activeLines = append(activeLines, colorizeTaskLine(line, task.status))
		activeTasks = append(activeTasks, task)
	}

	if final {
		r.tasks = nil
	} else {
		r.tasks = activeTasks
	}
	return finalLines, activeLines
}

func (r *interactiveRenderer) clearRenderedLines() {
	if r.renderedLines == 0 {
		return
	}

	_, _ = fmt.Fprintf(r.writer, "\x1b[%dA", r.renderedLines)
	for i := 0; i < r.renderedLines; i++ {
		_, _ = fmt.Fprint(r.writer, "\r\x1b[2K")
		if i < r.renderedLines-1 {
			_, _ = fmt.Fprint(r.writer, "\n")
		}
	}
	if r.renderedLines > 1 {
		_, _ = fmt.Fprintf(r.writer, "\x1b[%dA", r.renderedLines-1)
	}
	r.renderedLines = 0
}

func (t *interactiveTask) Update(message string) {
	t.renderer.mu.Lock()
	if t.status == taskRunning {
		t.message = message
	}
	t.renderer.mu.Unlock()
}

func (t *interactiveTask) Complete() {
	t.renderer.mu.Lock()
	if t.status == taskRunning {
		t.status = taskComplete
	}
	t.renderer.mu.Unlock()
}

func (t *interactiveTask) Error() {
	t.renderer.mu.Lock()
	if t.status == taskRunning {
		t.status = taskError
	}
	t.renderer.mu.Unlock()
}

func formatTaskLine(message string, status taskStatus, frame string, width int) string {
	prefix := frame + " "
	switch status {
	case taskComplete:
		prefix = "\u2713 "
	case taskError:
		prefix = "\u2717 "
	}

	message = strings.NewReplacer("\r", " ", "\n", " ").Replace(message)
	prefixWidth := runewidth.StringWidth(prefix)
	if width <= prefixWidth {
		return runewidth.Truncate(prefix, width, "")
	}
	maxMessageWidth := width - prefixWidth
	if maxMessageWidth < 0 {
		maxMessageWidth = 0
	}
	return prefix + runewidth.Truncate(message, maxMessageWidth, "")
}

func colorizeTaskLine(line string, status taskStatus) string {
	if line == "" {
		return line
	}

	color := ansiBrightGreen
	if status == taskError {
		color = ansiBrightRed
	}
	prefixEnd := strings.IndexByte(line, ' ')
	if prefixEnd < 0 {
		prefixEnd = len(line)
	}
	return color + line[:prefixEnd] + ansiReset + line[prefixEnd:]
}

type plainRenderer struct {
	mu     sync.Mutex
	writer io.Writer
}

type plainTask struct {
	renderer *plainRenderer
	mu       sync.Mutex
	message  string
	emitted  bool
}

func newPlainRenderer(writer io.Writer) *plainRenderer {
	return &plainRenderer{writer: writer}
}

func (r *plainRenderer) NewTask(message string) taskHandle {
	return &plainTask{renderer: r, message: message}
}

func (r *plainRenderer) Println(message string) {
	r.mu.Lock()
	_, _ = fmt.Fprintln(r.writer, message)
	r.mu.Unlock()
}

func (r *plainRenderer) BlankLine() {
	r.Println("")
}

func (r *plainRenderer) Stop() {}

func (t *plainTask) Update(message string) {
	t.mu.Lock()
	if !t.emitted {
		t.message = message
	}
	t.mu.Unlock()
}

func (t *plainTask) Complete() {
	t.emit()
}

func (t *plainTask) Error() {
	t.emit()
}

func (t *plainTask) emit() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.emitted {
		return
	}

	t.renderer.mu.Lock()
	_, _ = fmt.Fprintln(t.renderer.writer, t.message)
	t.renderer.mu.Unlock()
	t.emitted = true
}

type silentRenderer struct{}

type silentTask struct{}

func (silentRenderer) NewTask(string) taskHandle { return silentTask{} }
func (silentRenderer) Println(string)            {}
func (silentRenderer) BlankLine()                {}
func (silentRenderer) Stop()                     {}
func (silentTask) Update(string)                 {}
func (silentTask) Complete()                     {}
func (silentTask) Error()                        {}
