package logger

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"os"
)

var (
	Logger *log.Logger
)

func init() {
	styles := log.DefaultStyles()
	styles.Levels[log.FatalLevel] = lipgloss.NewStyle().SetString("FATAL!!").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "#9966CC", Dark: "#9966CC"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().SetString("ERROR!!").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "203", Dark: "203"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().SetString("INFO >>").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "45", Dark: "45"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().SetString("DEBUG ::").Padding(0, 1, 0, 1).Background(lipgloss.AdaptiveColor{Light: "75", Dark: "75"}).Foreground(lipgloss.Color("0")).Bold(true)
	styles.Keys["critical"] = lipgloss.NewStyle().Foreground(lipgloss.Color("#9966CC"))
	styles.Values["critical"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["hint"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["hint"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["status"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["status"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["object"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["object"] = lipgloss.NewStyle().Bold(true)
	styles.Keys["ready"] = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	styles.Values["ready"] = lipgloss.NewStyle().Bold(true)
	Logger = log.New(os.Stderr)
	Logger.SetStyles(styles)
	Logger.SetLevel(log.DebugLevel)
}

func ErrHandle(err error) {
	if err != nil {
		Logger.Fatal("Exit", "critical", err)
	}
}
