package lifecycle_test

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	arkerrors "github.com/goark-projects/goark/errors"
	"github.com/goark-projects/goark/lifecycle"
)

type testHook struct {
	name     string
	order    int
	log      *[]string
	startErr error
	stopErr  error
	closeErr error
}

func (h *testHook) Order() int {
	return h.order
}

func (h *testHook) Start(context.Context) error {
	*h.log = append(*h.log, "start:"+h.name)
	return h.startErr
}

func (h *testHook) Stop(context.Context) error {
	*h.log = append(*h.log, "stop:"+h.name)
	return h.stopErr
}

func (h *testHook) Close() error {
	*h.log = append(*h.log, "close:"+h.name)
	return h.closeErr
}

type priorityTestHook struct {
	*testHook
}

func (h *priorityTestHook) PriorityOrdered() {
}

func TestManager_whenStartedStoppedAndClosed_shouldRespectOrder(t *testing.T) {
	log := make([]string, 0, 6)
	manager := lifecycle.NewManager()
	if err := manager.Register("late", &testHook{name: "late", order: 20, log: &log}); err != nil {
		t.Fatalf("register late failed: %v", err)
	}
	if err := manager.Register("early", &testHook{name: "early", order: 10, log: &log}); err != nil {
		t.Fatalf("register early failed: %v", err)
	}

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !manager.Running() {
		t.Fatal("manager should be running")
	}
	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if manager.Running() {
		t.Fatal("manager should be stopped")
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	expected := []string{
		"start:early",
		"start:late",
		"stop:late",
		"stop:early",
		"close:late",
		"close:early",
	}
	if !reflect.DeepEqual(log, expected) {
		t.Fatalf("unexpected lifecycle order: %#v", log)
	}
}

func TestManager_whenHooksImplementPriorityOrdered_shouldUseGlobalOrderContracts(t *testing.T) {
	log := make([]string, 0, 6)
	manager := lifecycle.NewManager()
	if err := manager.Register("normal-early", &testHook{name: "normal-early", order: 10, log: &log}); err != nil {
		t.Fatalf("register normal early failed: %v", err)
	}
	if err := manager.Register("priority", &priorityTestHook{testHook: &testHook{name: "priority", order: 100, log: &log}}); err != nil {
		t.Fatalf("register priority failed: %v", err)
	}

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if err := manager.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	expected := []string{
		"start:priority",
		"start:normal-early",
		"stop:normal-early",
		"stop:priority",
		"close:normal-early",
		"close:priority",
	}
	if !reflect.DeepEqual(log, expected) {
		t.Fatalf("unexpected priority lifecycle order: %#v", log)
	}
}

func TestManager_whenStartFails_shouldRollbackStartedHooks(t *testing.T) {
	log := make([]string, 0, 3)
	manager := lifecycle.NewManager()
	if err := manager.Register("ok", &testHook{name: "ok", order: 10, log: &log}); err != nil {
		t.Fatalf("register ok failed: %v", err)
	}
	if err := manager.Register("fail", &testHook{name: "fail", order: 20, log: &log, startErr: stderrors.New("boom")}); err != nil {
		t.Fatalf("register fail failed: %v", err)
	}

	err := manager.Start(context.Background())
	if err == nil {
		t.Fatal("expected start error")
	}
	if !arkerrors.Is(err, arkerrors.CodeLifecycle) {
		t.Fatalf("expected lifecycle error, got %v", err)
	}
	if manager.Running() {
		t.Fatal("manager should not be running")
	}
	expected := []string{"start:ok", "start:fail", "stop:ok"}
	if !reflect.DeepEqual(log, expected) {
		t.Fatalf("unexpected rollback order: %#v", log)
	}
}

func TestManager_whenClosedWithoutStart_shouldCloseRegisteredHooks(t *testing.T) {
	log := make([]string, 0, 2)
	manager := lifecycle.NewManager()
	if err := manager.Register("late", &testHook{name: "late", order: 20, log: &log}); err != nil {
		t.Fatalf("register late failed: %v", err)
	}
	if err := manager.Register("early", &testHook{name: "early", order: 10, log: &log}); err != nil {
		t.Fatalf("register early failed: %v", err)
	}

	if err := manager.Close(context.Background()); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	expected := []string{"close:late", "close:early"}
	if !reflect.DeepEqual(log, expected) {
		t.Fatalf("unexpected close order: %#v", log)
	}
}

func TestManager_whenRegisteringWhileRunning_shouldReturnLifecycleError(t *testing.T) {
	log := make([]string, 0, 1)
	manager := lifecycle.NewManager()
	if err := manager.Register("hook", &testHook{name: "hook", log: &log}); err != nil {
		t.Fatalf("register hook failed: %v", err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	err := manager.Register("another", &testHook{name: "another", log: &log})
	if err == nil {
		t.Fatal("expected register error")
	}
	if !arkerrors.Is(err, arkerrors.CodeLifecycle) {
		t.Fatalf("expected lifecycle error, got %v", err)
	}
}
