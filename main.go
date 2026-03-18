package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ServiceStatus represents the health status of a service.
type ServiceStatus int

const (
	Healthy ServiceStatus = iota
	Degraded
	Down
)

func (s ServiceStatus) String() string {
	switch s {
	case Healthy:
		return "healthy"
	case Degraded:
		return "degraded"
	case Down:
		return "down"
	default:
		return "unknown"
	}
}

// StatusChange records a status transition.
type StatusChange struct {
	Timestamp time.Time     `json:"timestamp"`
	OldStatus ServiceStatus `json:"old_status"`
	NewStatus ServiceStatus `json:"new_status"`
}

// Service represents a monitored microservice.
type Service struct {
	Name       string         `json:"name"`
	Status     ServiceStatus  `json:"status"`
	LatencyP50 float64        `json:"latency_p50_ms"`
	LatencyP99 float64        `json:"latency_p99_ms"`
	ErrorRate  float64        `json:"error_rate"`
	Uptime     float64        `json:"uptime"`
	History    []StatusChange `json:"history"`
}

type viewMode int

const (
	dashboardView viewMode = iota
	detailView
	helpView
)

type tickMsg time.Time

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	healthyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	degradedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	downStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	headerStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	selectedStyle    = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236"))
	detailLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
)

var serviceNames = []string{
	"api-gateway",
	"auth-service",
	"user-service",
	"order-service",
	"payment-service",
	"inventory-service",
	"notification-service",
	"analytics-service",
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func initServices() []Service {
	services := make([]Service, len(serviceNames))
	for i, name := range serviceNames {
		services[i] = Service{
			Name:       name,
			Status:     Healthy,
			LatencyP50: 10 + rand.Float64()*40,
			LatencyP99: 50 + rand.Float64()*150,
			ErrorRate:  rand.Float64() * 0.5,
			Uptime:     99.5 + rand.Float64()*0.5,
			History:    []StatusChange{},
		}
	}
	return services
}

func randomizeServices(services []Service) []Service {
	for i := range services {
		// ~20% chance of status change per tick
		if rand.Float64() < 0.2 {
			oldStatus := services[i].Status
			r := rand.Float64()
			var newStatus ServiceStatus
			if r < 0.7 {
				newStatus = Healthy
			} else if r < 0.9 {
				newStatus = Degraded
			} else {
				newStatus = Down
			}
			if newStatus != oldStatus {
				change := StatusChange{
					Timestamp: time.Now(),
					OldStatus: oldStatus,
					NewStatus: newStatus,
				}
				services[i].History = append(services[i].History, change)
				if len(services[i].History) > 10 {
					services[i].History = services[i].History[len(services[i].History)-10:]
				}
				services[i].Status = newStatus
			}
		}
		// Update metrics based on current status
		services[i].LatencyP50 = 10 + rand.Float64()*40
		services[i].LatencyP99 = 50 + rand.Float64()*150
		switch services[i].Status {
		case Healthy:
			services[i].ErrorRate = rand.Float64() * 0.5
			services[i].Uptime = 99.5 + rand.Float64()*0.5
		case Degraded:
			services[i].ErrorRate = 1 + rand.Float64()*4
			services[i].Uptime = 95 + rand.Float64()*4
		case Down:
			services[i].ErrorRate = 50 + rand.Float64()*50
			services[i].Uptime = rand.Float64() * 50
		}
	}
	return services
}

type model struct {
	services []Service
	cursor   int
	view     viewMode
	width    int
	height   int
}

func initialModel() model {
	return model{
		services: initServices(),
		view:     dashboardView,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.services = randomizeServices(m.services)
		return m, tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.view == dashboardView && m.cursor < len(m.services)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.view == dashboardView && m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.view == dashboardView {
				m.view = detailView
			}
		case "esc", "backspace":
			if m.view != dashboardView {
				m.view = dashboardView
			}
		case "?":
			if m.view == helpView {
				m.view = dashboardView
			} else {
				m.view = helpView
			}
		}
	}
	return m, nil
}

func statusStyle(s ServiceStatus) lipgloss.Style {
	switch s {
	case Healthy:
		return healthyStyle
	case Degraded:
		return degradedStyle
	case Down:
		return downStyle
	default:
		return helpStyle
	}
}

func (m model) viewDashboard() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Microservice Health Monitor"))
	b.WriteString("\n\n")

	header := fmt.Sprintf("  %-24s %-10s %10s %10s %10s %10s",
		"SERVICE", "STATUS", "P50(ms)", "P99(ms)", "ERR(%)", "UPTIME(%)")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 80))
	b.WriteString("\n")

	for i, svc := range m.services {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		statusStr := statusStyle(svc.Status).Render(fmt.Sprintf("%-10s", svc.Status.String()))
		line := fmt.Sprintf("%s%-24s %s %10.1f %10.1f %10.2f %10.2f",
			cursor, svc.Name, statusStr, svc.LatencyP50, svc.LatencyP99, svc.ErrorRate, svc.Uptime)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate  enter: detail  ?: help  q: quit"))
	return b.String()
}

func (m model) viewDetail() string {
	svc := m.services[m.cursor]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Service Detail: %s", svc.Name)))
	b.WriteString("\n\n")

	b.WriteString(detailLabelStyle.Render("Status:      "))
	b.WriteString(statusStyle(svc.Status).Render(svc.Status.String()))
	b.WriteString("\n")
	b.WriteString(detailLabelStyle.Render(fmt.Sprintf("Latency P50: %.1f ms\n", svc.LatencyP50)))
	b.WriteString(detailLabelStyle.Render(fmt.Sprintf("Latency P99: %.1f ms\n", svc.LatencyP99)))
	b.WriteString(detailLabelStyle.Render(fmt.Sprintf("Error Rate:  %.2f%%\n", svc.ErrorRate)))
	b.WriteString(detailLabelStyle.Render(fmt.Sprintf("Uptime:      %.2f%%\n", svc.Uptime)))

	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Last Status Changes (most recent first)"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 60))
	b.WriteString("\n")

	if len(svc.History) == 0 {
		b.WriteString(helpStyle.Render("  No status changes recorded yet"))
		b.WriteString("\n")
	} else {
		for i := len(svc.History) - 1; i >= 0; i-- {
			h := svc.History[i]
			b.WriteString(fmt.Sprintf("  %s  %s -> %s\n",
				h.Timestamp.Format("15:04:05"),
				statusStyle(h.OldStatus).Render(h.OldStatus.String()),
				statusStyle(h.NewStatus).Render(h.NewStatus.String()),
			))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc: back  ?: help  q: quit"))
	return b.String()
}

func (m model) viewHelp() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Help"))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 40))
	b.WriteString("\n")
	b.WriteString("  j / down    Move cursor down\n")
	b.WriteString("  k / up      Move cursor up\n")
	b.WriteString("  enter       View service detail\n")
	b.WriteString("  esc         Back to dashboard\n")
	b.WriteString("  ?           Toggle help\n")
	b.WriteString("  q / ctrl+c  Quit\n")
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Status Legend"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", 40))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  Service operating normally\n", healthyStyle.Render("healthy")))
	b.WriteString(fmt.Sprintf("  %s  Service experiencing issues\n", degradedStyle.Render("degraded")))
	b.WriteString(fmt.Sprintf("  %s  Service is unreachable\n", downStyle.Render("down")))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc/?: back  q: quit"))
	return b.String()
}

func (m model) View() string {
	switch m.view {
	case detailView:
		return m.viewDetail()
	case helpView:
		return m.viewHelp()
	default:
		return m.viewDashboard()
	}
}

func main() {
	jsonFlag := flag.Bool("json", false, "Output service status as JSON and exit")
	flag.Parse()

	if *jsonFlag {
		services := initServices()
		data, err := json.MarshalIndent(services, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
