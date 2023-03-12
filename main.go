package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gen2brain/beeep"
)

type model struct {
	timer             timer.Model
	keymap            keymap
	help              help.Model
	onBreak           bool
	quitting          bool
	userWorkDuration  time.Duration
	userBreakDuration time.Duration
	userPeriods       int
	currentPeriod     int
}

type keymap struct {
	start key.Binding
	stop  key.Binding
	reset key.Binding
	quit  key.Binding
}

func (m model) Init() tea.Cmd {
	return m.timer.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keymap.reset):
			if !m.onBreak {
				m.timer.Timeout = m.userWorkDuration
			} else {
				m.timer.Timeout = m.userBreakDuration
			}
		case key.Matches(msg, m.keymap.start, m.keymap.stop):
			return m, m.timer.Toggle()
		}
	}

	if m.currentPeriod > m.userPeriods {
		m.quitting = true
		return m, tea.Quit
	}
	if _, ok := msg.(timer.TimeoutMsg); ok {
		if !m.onBreak {
			m.onBreak = true
			err := beeep.Notify("Pom", "You are on break!", "asserts/information.png")
			if err != nil {
				fmt.Println("Something went wrong, a notification was not issued...")
			}
			m.timer.Timeout = m.userBreakDuration
			m.timer.Start()
			return m, nil
		} else {
			m.onBreak = false
			err := beeep.Notify("Pom", "Start working...", "asserts/information.png")
			if err != nil {
				fmt.Println("Something went wrong, a notification was not issued...")
			}
			m.timer.Timeout = m.userWorkDuration
			m.timer.Start()
			m.currentPeriod++
			return m, nil
		}
	}

	if !m.onBreak {
		return workUpdates(msg, m)
	} else {
		return breakUpdates(msg, m)
	}
}

func (m model) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.start,
		m.keymap.stop,
		m.keymap.reset,
		m.keymap.quit,
	})
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!"
	}

	if !m.onBreak {
		return workView(m)
	} else {
		return breakView(m)
	}
}

func workUpdates(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case timer.StartStopMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		m.keymap.stop.SetEnabled(m.timer.Running())
		m.keymap.start.SetEnabled(!m.timer.Running())
		return m, cmd

	case timer.TimeoutMsg:
		m.onBreak = true
	}

	return m, nil
}

func breakUpdates(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case timer.StartStopMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		m.keymap.stop.SetEnabled(m.timer.Running())
		m.keymap.start.SetEnabled(!m.timer.Running())
		return m, cmd

	case timer.TimeoutMsg:
		m.onBreak = false
	}

	return m, nil
}

func workView(m model) string {
	s := m.timer.View()
	s += "\n"
	s = "Relaxing in " + s
	s += "\n"
	s += fmt.Sprintf("%d/%d", m.currentPeriod, m.userPeriods)
	s += m.helpView()
	return s
}

func breakView(m model) string {
	s := m.timer.View()
	s += "\n"
	s = "Working in " + s
	s += "\n"
	s += fmt.Sprintf("%d/%d", m.currentPeriod, m.userPeriods)
	s += m.helpView()
	return s
}

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("Please enter - 1: work duration, 2: break duration, 3: number of work+break blocks to do.\n")
		os.Exit(1)
	}
	inputWorkDuration, err := strconv.Atoi(os.Args[1])
	if err != nil {
		// TODO add a better error message with example usage
		fmt.Printf("Timout not set correctly, needs to be an int\n")
		os.Exit(1)
	}

	inputBreakDuration, err := strconv.Atoi(os.Args[2])
	if err != nil {
		// TODO add a better error message with example usage
		fmt.Printf("Break duration not set correctly, needs to be an int\n")
		os.Exit(1)
	}

	// TODO this does not catch if user does not input anything (error with os.Args not strconv!)
	inputPeriod, err := strconv.Atoi(os.Args[3])
	if err != nil {
		// TODO add a better error message with example usage
		fmt.Printf("Work periods not set correctly, please input an int")
		os.Exit(1)
	}

	workDuration := time.Duration(inputWorkDuration) * time.Second
	breakDuration := time.Duration(inputBreakDuration) * time.Second
	m := model{
		timer: timer.NewWithInterval(workDuration, time.Millisecond),
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start"),
			),
			stop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "pause"),
			),
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
			),
			quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
		},
		userWorkDuration:  workDuration,
		userBreakDuration: breakDuration,
		userPeriods:       inputPeriod,
		currentPeriod:     0,
		help:              help.NewModel(),
	}
	m.keymap.start.SetEnabled(false)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Uh oh, we encountered an error:", err)
		os.Exit(1)
	}
}
