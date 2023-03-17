package main

import (
	"fmt"
	"github.com/chelnak/ysmrr"
	"os"
	"strings"
)

type TaskManager struct {
	sm    ysmrr.SpinnerManager
	isOut bool
}

type Task struct {
	spinner *ysmrr.Spinner
	manager *TaskManager
	title   string
}

func InitTaskManager(isOut bool) *TaskManager {
	tm := &TaskManager{sm: ysmrr.NewSpinnerManager(), isOut: isOut}
	if isOut {
		tm.sm.Start()
	}
	return tm
}

func (tm *TaskManager) Reset() {
	if tm.isOut {
		tm.sm.Stop()
		tm.sm = ysmrr.NewSpinnerManager()
		tm.sm.Start()
	}
}

func (tm *TaskManager) Stop() {
	if tm.isOut {
		tm.sm.Stop()
	}
}

func (tm *TaskManager) Println(message string) {
	if tm.isOut {
		context := &Task{manager: tm}
		context.spinner = tm.sm.AddSpinner(message)
		context.Complete()
	}
}

func (tm *TaskManager) Run(title string, callback func(task *Task)) {
	context := &Task{manager: tm, title: title}
	if tm.isOut {
		context.spinner = tm.sm.AddSpinner(title)
	}
	callback(context)
}

func (tm *TaskManager) AsyncRun(title string, callback func(task *Task)) {
	context := &Task{manager: tm, title: title}
	if tm.isOut {
		context.spinner = tm.sm.AddSpinner(title)
	}
	go callback(context)
}

func (t *Task) Complete() {
	if t.spinner == nil {
		return
	}
	t.spinner.Complete()
}

func (t *Task) Println(message string) {
	if t.spinner == nil {
		return
	}
	t.spinner.UpdateMessage(message)
}

func (t *Task) Printf(format string, a ...interface{}) {
	if t.spinner == nil {
		return
	}
	t.spinner.UpdateMessagef(format, a...)
}

func (t *Task) CheckError(err error) {
	if err != nil {
		if t.spinner != nil {
			t.Printf("Fatal: %s, err: %v", strings.ToLower(t.title), err)
			t.spinner.Error()
			t.manager.Stop()
		} else {
			fmt.Printf("Fatal: %s, err: %v", strings.ToLower(t.title), err)
		}
		os.Exit(1)
	}
}
