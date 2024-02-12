package charm

import (
	"fmt"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"os"
)

const (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")
)

var (
	re = lipgloss.NewRenderer(os.Stdout)
	// HeaderStyle is the lipgloss style used for the table headers.
	HeaderStyle = re.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
	// CellStyle is the base lipgloss style used for the table rows.
	CellStyle = re.NewStyle().Padding(0, 2).Width(14)
	// OddRowStyle is the lipgloss style used for odd-numbered table rows.
	OddRowStyle = CellStyle.Copy().Foreground(gray)
	// EvenRowStyle is the lipgloss style used for even-numbered table rows.
	EvenRowStyle = CellStyle.Copy().Foreground(gray)
	// BorderStyle is the lipgloss style used for the table border.
	BorderStyle = lipgloss.NewStyle().Foreground(purple)
)

func CreateOptionsFromStrings(strings []string) []huh.Option[string] {
	var options []huh.Option[string]
	for _, str := range strings {
		options = append(options, huh.NewOption(string(str), string(str)))
	}
	return options
}

func GetForm[T comparable](selects ...*huh.Select[T]) *huh.Form {
	var fields []huh.Field
	for _, sel := range selects {
		fields = append(fields, sel)
	}
	group := huh.NewGroup(fields...)
	return huh.NewForm(group)
}

func CreateObjectArray(ObjectList [][]string, headers []string) {
	defaultHeaders := []string{"NAME", "STATUS", "READY", "MESSAGE"}
	if len(headers) == 0 {
		headers = defaultHeaders
	}
	styleFunc := func(row int, col int) lipgloss.Style {
		var style lipgloss.Style

		switch {
		case row == 0:
			return HeaderStyle
		default:
			style = EvenRowStyle
		}
		style = style.Copy().Width(0) // force to 0 width, will size automatically with wider element
		return style
	}
	t := table.New().Border(lipgloss.ThickBorder()).BorderStyle(BorderStyle).StyleFunc(styleFunc).Headers(headers...).Rows(ObjectList...)
	fmt.Println(t)
}
