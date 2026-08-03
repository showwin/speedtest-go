package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestInteractiveRendererCoalescesUpdates(t *testing.T) {
	var output bytes.Buffer
	renderer := newInteractiveRenderer(
		&output,
		[]string{"."},
		time.Hour,
		func() int { return defaultTerminalWidth },
	)

	task := renderer.NewTask("starting")
	task.Update("first update")
	task.Update("final update")
	task.Complete()
	renderer.Stop()
	renderer.Stop()

	rendered := output.String()
	if strings.Contains(rendered, "first update") {
		t.Fatalf("intermediate update was rendered: %q", rendered)
	}
	if count := strings.Count(rendered, "final update"); count != 1 {
		t.Fatalf("final update rendered %d times: %q", count, rendered)
	}
}

func TestInteractiveRendererSerializesBlankLineAfterCompletedTask(t *testing.T) {
	var output bytes.Buffer
	renderer := &interactiveRenderer{
		writer: &output,
		frames: []string{"."},
		width:  func() int { return defaultTerminalWidth },
	}

	task := renderer.NewTask("Retrieving Servers")
	renderer.render(false)
	task.Update("Found 30 Public Servers")
	task.Complete()
	renderer.BlankLine()
	renderer.render(false)

	rendered := output.String()
	want := "\x1b[1A\r\x1b[2K\x1b[92m\u2713\x1b[0m Found 30 Public Servers\n\n"
	if !strings.Contains(rendered, want) {
		t.Fatalf("completed task and blank line were not serialized together: %q", rendered)
	}
	if strings.Contains(rendered, "\u2713 \n") {
		t.Fatalf("blank line was rendered as a completed task: %q", rendered)
	}
}

func TestInteractiveRendererColorsStatusSymbol(t *testing.T) {
	if got, want := colorizeTaskLine(". running", taskRunning), "\x1b[92m.\x1b[0m running"; got != want {
		t.Fatalf("unexpected running style: got %q, want %q", got, want)
	}
	if got, want := colorizeTaskLine("\u2713 done", taskComplete), "\x1b[92m\u2713\x1b[0m done"; got != want {
		t.Fatalf("unexpected complete style: got %q, want %q", got, want)
	}
	if got, want := colorizeTaskLine("\u2717 failed", taskError), "\x1b[91m\u2717\x1b[0m failed"; got != want {
		t.Fatalf("unexpected error style: got %q, want %q", got, want)
	}
}

func TestInteractiveRendererSanitizesAndTruncatesMessages(t *testing.T) {
	line := formatTaskLine("hello\nworld", taskComplete, ".", 9)
	if line != "\u2713 hello w" {
		t.Fatalf("unexpected task line: %q", line)
	}

	wideLine := formatTaskLine("\u6d4b\u8bd5\u7f51\u7edc", taskComplete, ".", 6)
	if wideLine != "\u2713 \u6d4b\u8bd5" {
		t.Fatalf("unexpected wide task line: %q", wideLine)
	}

	if narrowLine := formatTaskLine("ignored", taskComplete, ".", 1); narrowLine != "\u2713" {
		t.Fatalf("unexpected narrow task line: %q", narrowLine)
	}
}

func TestPlainRendererEmitsFinalMessageOnce(t *testing.T) {
	var output bytes.Buffer
	renderer := newPlainRenderer(&output)
	task := renderer.NewTask("starting")

	task.Update("done")
	task.Complete()
	task.Complete()
	renderer.BlankLine()
	renderer.Stop()

	if got, want := output.String(), "done\n\n"; got != want {
		t.Fatalf("unexpected plain output: got %q, want %q", got, want)
	}
}

func TestSilentRendererProducesNoTaskOutput(t *testing.T) {
	renderer := silentRenderer{}
	task := renderer.NewTask("starting")

	task.Update("done")
	task.Complete()
	renderer.Println("ignored")
	renderer.BlankLine()
	renderer.Stop()
}

func TestTaskManagerStopWaitsForAsyncTasks(t *testing.T) {
	renderer := &testTaskRenderer{stopped: make(chan struct{})}
	manager := &TaskManager{renderer: renderer}
	started := make(chan struct{})
	release := make(chan struct{})
	manager.AsyncRun("async", func(task *Task) {
		close(started)
		<-release
		task.Complete()
	})
	<-started

	stopped := make(chan struct{})
	go func() {
		manager.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("manager stopped before its asynchronous task completed")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("manager did not stop after its asynchronous task completed")
	}
}

type testTaskRenderer struct {
	stopped chan struct{}
}

func (r *testTaskRenderer) NewTask(string) taskHandle { return silentTask{} }
func (r *testTaskRenderer) Println(string)            {}
func (r *testTaskRenderer) BlankLine()                {}
func (r *testTaskRenderer) Stop()                     { close(r.stopped) }
