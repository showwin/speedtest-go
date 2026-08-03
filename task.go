package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

type TaskManager struct {
	renderer taskRenderer
	silent   bool
	async    sync.WaitGroup
}

type Task struct {
	handle  taskHandle
	manager *TaskManager
	title   string
}

func InitTaskManager(jsonOutput, unixOutput bool) *TaskManager {
	return &TaskManager{
		renderer: newTaskRenderer(jsonOutput, unixOutput),
		silent:   jsonOutput && !unixOutput,
	}
}

func (tm *TaskManager) Stop() {
	if tm == nil {
		return
	}
	tm.async.Wait()
	tm.stopRenderer()
}

func (tm *TaskManager) stopRenderer() {
	if tm.renderer == nil {
		return
	}
	tm.renderer.Stop()
}

func (tm *TaskManager) Println(message string) {
	if tm == nil || tm.renderer == nil {
		return
	}
	tm.renderer.Println(message)
}

func (tm *TaskManager) BlankLine() {
	if tm == nil || tm.renderer == nil {
		return
	}
	tm.renderer.BlankLine()
}

func (tm *TaskManager) RunWithTrigger(enable bool, title string, callback func(task *Task)) {
	if enable {
		tm.Run(title, callback)
	}
}

func (tm *TaskManager) Run(title string, callback func(task *Task)) {
	task := tm.newTask(title)
	callback(task)
}

func (tm *TaskManager) AsyncRun(title string, callback func(task *Task)) {
	task := tm.newTask(title)
	tm.async.Add(1)
	go func() {
		defer tm.async.Done()
		callback(task)
	}()
}

func (tm *TaskManager) newTask(title string) *Task {
	return &Task{
		handle:  tm.renderer.NewTask(title),
		manager: tm,
		title:   title,
	}
}

func (t *Task) Complete() {
	if t == nil || t.handle == nil {
		return
	}
	t.handle.Complete()
}

func (t *Task) Updatef(format string, a ...interface{}) {
	if t == nil || t.handle == nil {
		return
	}
	t.handle.Update(fmt.Sprintf(format, a...))
}

func (t *Task) Update(message string) {
	if t == nil || t.handle == nil {
		return
	}
	t.handle.Update(message)
}

func (t *Task) Println(message string) {
	t.Update(message)
}

func (t *Task) Printf(format string, a ...interface{}) {
	t.Update(fmt.Sprintf(format, a...))
}

func (t *Task) CheckError(err error) {
	if err == nil {
		return
	}

	message := fmt.Sprintf("Fatal: %s, err: %v", strings.ToLower(t.title), err)
	if t.handle != nil {
		t.handle.Update(message)
		t.handle.Error()
	}
	if t.manager.silent {
		_, _ = fmt.Fprintln(os.Stderr, message)
	}
	t.manager.stopRenderer()
	os.Exit(1)
}
