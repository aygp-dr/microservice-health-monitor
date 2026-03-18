package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestServiceStatusString(t *testing.T) {
	tests := []struct {
		status ServiceStatus
		want   string
	}{
		{Healthy, "healthy"},
		{Degraded, "degraded"},
		{Down, "down"},
		{ServiceStatus(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("ServiceStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestInitServices(t *testing.T) {
	services := initServices()
	if len(services) != 8 {
		t.Fatalf("initServices() returned %d services, want 8", len(services))
	}

	expectedNames := []string{
		"api-gateway", "auth-service", "user-service", "order-service",
		"payment-service", "inventory-service", "notification-service", "analytics-service",
	}

	for i, svc := range services {
		if svc.Name != expectedNames[i] {
			t.Errorf("service[%d].Name = %q, want %q", i, svc.Name, expectedNames[i])
		}
		if svc.Status != Healthy {
			t.Errorf("service %s initial status = %v, want Healthy", svc.Name, svc.Status)
		}
		if svc.LatencyP50 < 10 || svc.LatencyP50 > 50 {
			t.Errorf("service %s P50 = %.1f, want [10, 50]", svc.Name, svc.LatencyP50)
		}
		if svc.LatencyP99 < 50 || svc.LatencyP99 > 200 {
			t.Errorf("service %s P99 = %.1f, want [50, 200]", svc.Name, svc.LatencyP99)
		}
		if svc.ErrorRate < 0 || svc.ErrorRate > 0.5 {
			t.Errorf("service %s error rate = %.2f, want [0, 0.5]", svc.Name, svc.ErrorRate)
		}
		if svc.Uptime < 99.5 || svc.Uptime > 100 {
			t.Errorf("service %s uptime = %.2f, want [99.5, 100]", svc.Name, svc.Uptime)
		}
		if len(svc.History) != 0 {
			t.Errorf("service %s should have empty history", svc.Name)
		}
	}
}

func TestRandomizeServicesHistoryCap(t *testing.T) {
	// Pre-fill history to 10 entries
	history := make([]StatusChange, 10)
	for i := range history {
		history[i] = StatusChange{
			Timestamp: time.Now(),
			OldStatus: Healthy,
			NewStatus: Degraded,
		}
	}
	services := []Service{{
		Name:    "test-service",
		Status:  Healthy,
		History: history,
	}}

	// Run many times to trigger status changes
	for i := 0; i < 200; i++ {
		services = randomizeServices(services)
		if len(services[0].History) > 10 {
			t.Fatalf("iteration %d: history length %d exceeds cap of 10", i, len(services[0].History))
		}
	}
}

func TestRandomizeServicesMetrics(t *testing.T) {
	services := []Service{
		{Name: "svc-healthy", Status: Healthy},
		{Name: "svc-degraded", Status: Degraded},
		{Name: "svc-down", Status: Down},
	}

	// Run once to set metrics
	services = randomizeServices(services)

	for _, svc := range services {
		if svc.LatencyP50 < 10 || svc.LatencyP50 > 50 {
			t.Errorf("%s P50 = %.1f, out of range [10, 50]", svc.Name, svc.LatencyP50)
		}
		if svc.LatencyP99 < 50 || svc.LatencyP99 > 200 {
			t.Errorf("%s P99 = %.1f, out of range [50, 200]", svc.Name, svc.LatencyP99)
		}
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()
	if len(m.services) != 8 {
		t.Errorf("initialModel has %d services, want 8", len(m.services))
	}
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
	if m.view != dashboardView {
		t.Errorf("initial view = %d, want dashboardView", m.view)
	}
}

func TestModelInit(t *testing.T) {
	m := initialModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a tick command")
	}
}

func keyMsg(s string) tea.Msg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestModelUpdate_NavigateDown(t *testing.T) {
	m := initialModel()

	updated, _ := m.Update(keyMsg("j"))
	m = updated.(model)
	if m.cursor != 1 {
		t.Errorf("after j, cursor = %d, want 1", m.cursor)
	}
}

func TestModelUpdate_NavigateUp(t *testing.T) {
	m := initialModel()
	m.cursor = 3

	updated, _ := m.Update(keyMsg("k"))
	m = updated.(model)
	if m.cursor != 2 {
		t.Errorf("after k, cursor = %d, want 2", m.cursor)
	}
}

func TestModelUpdate_NavigateBounds(t *testing.T) {
	m := initialModel()

	// Can't go above 0
	updated, _ := m.Update(keyMsg("k"))
	m = updated.(model)
	if m.cursor != 0 {
		t.Errorf("cursor went below 0: %d", m.cursor)
	}

	// Move to last service
	for i := 0; i < 10; i++ {
		updated, _ = m.Update(keyMsg("j"))
		m = updated.(model)
	}
	if m.cursor != 7 {
		t.Errorf("cursor should cap at 7, got %d", m.cursor)
	}
}

func TestModelUpdate_EnterDetail(t *testing.T) {
	m := initialModel()

	updated, _ := m.Update(keyMsg("enter"))
	m = updated.(model)
	if m.view != detailView {
		t.Errorf("after enter, view = %d, want detailView", m.view)
	}
}

func TestModelUpdate_EscBack(t *testing.T) {
	m := initialModel()
	m.view = detailView

	updated, _ := m.Update(keyMsg("esc"))
	m = updated.(model)
	if m.view != dashboardView {
		t.Errorf("after esc, view = %d, want dashboardView", m.view)
	}
}

func TestModelUpdate_BackspaceBack(t *testing.T) {
	m := initialModel()
	m.view = detailView

	updated, _ := m.Update(keyMsg("backspace"))
	m = updated.(model)
	if m.view != dashboardView {
		t.Errorf("after backspace, view = %d, want dashboardView", m.view)
	}
}

func TestModelUpdate_HelpToggle(t *testing.T) {
	m := initialModel()

	updated, _ := m.Update(keyMsg("?"))
	m = updated.(model)
	if m.view != helpView {
		t.Errorf("after ?, view = %d, want helpView", m.view)
	}

	updated, _ = m.Update(keyMsg("?"))
	m = updated.(model)
	if m.view != dashboardView {
		t.Errorf("after second ?, view = %d, want dashboardView", m.view)
	}
}

func TestModelUpdate_NavigationDisabledInDetailView(t *testing.T) {
	m := initialModel()
	m.view = detailView
	m.cursor = 0

	updated, _ := m.Update(keyMsg("j"))
	m = updated.(model)
	if m.cursor != 0 {
		t.Errorf("j should not move cursor in detail view, cursor = %d", m.cursor)
	}
}

func TestModelUpdate_Tick(t *testing.T) {
	m := initialModel()
	updated, cmd := m.Update(tickMsg(time.Now()))
	m = updated.(model)

	if cmd == nil {
		t.Error("tick should return a new tick command")
	}
	if len(m.services) != 8 {
		t.Errorf("after tick, services count = %d, want 8", len(m.services))
	}
}

func TestModelUpdate_WindowSize(t *testing.T) {
	m := initialModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(model)
	if m.width != 120 || m.height != 40 {
		t.Errorf("window size = %dx%d, want 120x40", m.width, m.height)
	}
}

func TestViewDashboard(t *testing.T) {
	m := initialModel()
	output := m.viewDashboard()

	if !strings.Contains(output, "Microservice Health Monitor") {
		t.Error("dashboard should contain title")
	}
	for _, name := range serviceNames {
		if !strings.Contains(output, name) {
			t.Errorf("dashboard should contain service name %q", name)
		}
	}
	if !strings.Contains(output, "healthy") {
		t.Error("dashboard should show status")
	}
	if !strings.Contains(output, "P50") {
		t.Error("dashboard should show P50 header")
	}
	if !strings.Contains(output, "P99") {
		t.Error("dashboard should show P99 header")
	}
}

func TestViewDetail(t *testing.T) {
	m := initialModel()
	m.services[0].History = []StatusChange{
		{Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), OldStatus: Healthy, NewStatus: Degraded},
		{Timestamp: time.Date(2026, 1, 1, 12, 5, 0, 0, time.UTC), OldStatus: Degraded, NewStatus: Down},
	}
	output := m.viewDetail()

	if !strings.Contains(output, "api-gateway") {
		t.Error("detail view should show service name")
	}
	if !strings.Contains(output, "Latency P50") {
		t.Error("detail view should show latency")
	}
	if !strings.Contains(output, "12:00:00") {
		t.Error("detail view should show status change timestamps")
	}
	if !strings.Contains(output, "12:05:00") {
		t.Error("detail view should show second status change timestamp")
	}
}

func TestViewDetailEmpty(t *testing.T) {
	m := initialModel()
	output := m.viewDetail()
	if !strings.Contains(output, "No status changes recorded") {
		t.Error("detail view with no history should show empty message")
	}
}

func TestViewHelp(t *testing.T) {
	m := initialModel()
	output := m.viewHelp()

	if !strings.Contains(output, "Help") {
		t.Error("help view should contain title")
	}
	if !strings.Contains(output, "enter") {
		t.Error("help view should document enter key")
	}
	if !strings.Contains(output, "healthy") {
		t.Error("help view should show status legend")
	}
}

func TestViewRouting(t *testing.T) {
	m := initialModel()

	m.view = dashboardView
	if !strings.Contains(m.View(), "Microservice Health Monitor") {
		t.Error("View() should route to dashboard")
	}

	m.view = detailView
	if !strings.Contains(m.View(), "Service Detail") {
		t.Error("View() should route to detail")
	}

	m.view = helpView
	if !strings.Contains(m.View(), "Help") {
		t.Error("View() should route to help")
	}
}

func TestJSONSerialization(t *testing.T) {
	services := initServices()
	data, err := json.Marshal(services)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded []Service
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if len(decoded) != 8 {
		t.Errorf("JSON decoded %d services, want 8", len(decoded))
	}
	if decoded[0].Name != "api-gateway" {
		t.Errorf("first service name = %q, want api-gateway", decoded[0].Name)
	}
}

func TestStatusStyle(t *testing.T) {
	// Verify each status returns a distinct style (no panics)
	_ = statusStyle(Healthy)
	_ = statusStyle(Degraded)
	_ = statusStyle(Down)
	_ = statusStyle(ServiceStatus(99))
}
