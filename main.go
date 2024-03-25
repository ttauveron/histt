package main

import (
	"bufio"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"os"
	"regexp"
	"strings"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sys/unix"
)

const headerSize = 6

type Mode int

const (
	ExactMatching Mode = iota
	Keywords
	Regex
)

func (m Mode) String() string {
	switch m {
	case ExactMatching:
		return "exact"
	case Keywords:
		return "keywords"
	case Regex:
		return "regex"
	default:
		return "unknown"
	}
}

type TextCase int

const (
	Insensitive TextCase = iota
	Sensitive
)

func (c TextCase) String() string {
	switch c {
	case Insensitive:
		return "insensitive"
	case Sensitive:
		return "sensitive"
	default:
		return "unknown"
	}
}

type model struct {
	commands    []string // All commands from history
	filtered    []string // Filtered commands based on query
	query       string   // Current user input for filtering
	selected    int      // Currently selected command index
	viewStart   int      // Index in `filtered` where the view starts
	viewEnd     int      // Index in `filtered` where the view ends
	displaySize int      // Number of commands to display at a time
	textInput   textinput.Model
	width       int
	height      int
	mode        Mode
	textCase    TextCase
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Command..."
	ti.Focus()
	ti.Prompt = ">> "
	ti.CharLimit = 10000

	// Assuming we want to display 10 commands at a time
	displaySize := 20

	history, _ := readHistory(os.Getenv("HOME") + "/.bash_history")
	return model{
		commands:    history,
		filtered:    history,
		selected:    0,
		viewStart:   0,
		viewEnd:     displaySize,
		displaySize: displaySize,
		textInput:   ti,
	}
}

func fillTerminalInput(cmd string, padding bool) {
	if cmd == "" {
		return
	}

	fd := int(os.Stdin.Fd())
	for _, c := range cmd {
		_, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(fd),
			uintptr(unix.TIOCSTI),
			uintptr(unsafe.Pointer(&c)),
		)
		if errno != 0 {
			fmt.Fprintf(os.Stderr, "Failed to simulate terminal input: %v\n", errno)
			return
		}
	}

	if padding {
		fmt.Println()
	}
}

func removeDuplicates(elements []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, element := range elements {
		if _, found := seen[element]; !found {
			seen[element] = true
			result = append(result, element)
		}
	}

	return result
}

func removeComments(elements []string) []string {
	var result []string

	for _, element := range elements {
		if !strings.HasPrefix(element, "#") {
			result = append(result, element)
		}
	}

	return result
}

func readHistory(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var commands []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := scanner.Text()
		commands = append(commands, command)
	}

	for i, j := 0, len(commands)-1; i < j; i, j = i+1, j-1 {
		commands[i], commands[j] = commands[j], commands[i]
	}

	results := removeComments(commands)
	return removeDuplicates(results), scanner.Err()
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) filterCommands() {
	var query string
	if m.textCase == Insensitive {
		query = strings.ToLower(m.query)
	} else {
		query = m.query
	}

	var filtered []string
	for _, cmd := range m.commands {
		var cmdTextSearch string
		if m.textCase == Insensitive {
			cmdTextSearch = strings.ToLower(cmd) // Convert to lowercase for case-insensitive comparison.
		} else {
			cmdTextSearch = cmd
		}

		switch m.mode {
		case ExactMatching:
			if strings.HasPrefix(cmdTextSearch, query) {
				filtered = append(filtered, cmd)
			}
		case Keywords:
			matches := true
			for _, word := range strings.Split(query, " ") {
				if !strings.Contains(cmdTextSearch, word) {
					matches = false
					break
				}
			}
			if matches {
				filtered = append(filtered, cmd)
			}
		case Regex:
			matched, err := regexp.MatchString(query, cmdTextSearch)
			if err == nil && matched {
				filtered = append(filtered, cmd)
			}

		default:
			filtered = append(filtered, cmd)
		}
	}
	m.filtered = filtered
	// Reset view and selection
	m.viewStart = 0
	m.selected = 0
	m.viewEnd = m.height - headerSize
	if m.viewEnd > len(m.filtered) {
		m.viewEnd = len(m.filtered)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyUp:
			if m.selected > 0 {
				m.selected--
				if m.selected < m.viewStart {
					m.viewStart--
					m.viewEnd--
				}
			}
		case tea.KeyDown:
			if m.selected < len(m.filtered)-1 {
				m.selected++
				if m.selected >= m.viewEnd {
					m.viewStart++
					m.viewEnd++
				}
			}

		case tea.KeyTab:
			// fillTerminalInput(m.commands[m.selected],true)
			return m, tea.Quit

		case tea.KeyCtrlE:
			m.switchMode()
			m.query = m.textInput.Value()
			m.filterCommands()
			//return m, nil

		case tea.KeyCtrlT:
			m.switchCase()
			m.query = m.textInput.Value()
			m.filterCommands()
			//return m, nil

		default:
			m.query = m.textInput.Value()
			m.filterCommands()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewEnd = m.height - headerSize
	}

	return m, cmd
}

func (m *model) switchMode() {
	m.mode = (m.mode + 1) % 3
}

func (m *model) switchCase() {
	m.textCase = (m.textCase + 1) % 2
}

func fitStringToWidth(s string, width int) string {
	if len(s) <= width || width < 10 {
		return s
	}

	partLength := (width - 3) / 2 // Subtract 3 for the ellipsis and divide by 2 for two parts.
	return s[:partLength] + "..." + s[len(s)-partLength:]
}

func (m model) View() string {

	var b strings.Builder
	b.WriteString(m.textInput.View())
	b.WriteString(fmt.Sprintf("\n- HISTORY - match:%s (C-e) - case:%s (C-t)", m.mode, m.textCase))
	b.WriteString("\n\n")

	displayEnd := min(m.viewEnd, len(m.filtered))
	for i, cmd := range m.filtered[m.viewStart:displayEnd] {
		cursor := " " // Not selected
		if m.viewStart+i == m.selected {
			cursor = ">"
		}

		cmdDisplay := fitStringToWidth(cmd, m.width-2)
		b.WriteString(fmt.Sprintf("%s %s\n", cursor, cmdDisplay))
	}

	b.WriteString("\nPress q to quit.\n")
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	// Assert the finalModel back to your specific model type to access its fields.
	if m, ok := finalModel.(model); ok {
		fillTerminalInput(m.filtered[m.selected], true)
	} else {
		fmt.Println("Could not retrieve the final model.")
	}
}
