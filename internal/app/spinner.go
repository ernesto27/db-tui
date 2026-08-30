package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

const spinnerInterval = 100 * time.Millisecond

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type spinnerTickMsg struct{}

func spinnerTick() tea.Cmd {
	return tea.Tick(spinnerInterval, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m *Model) startSpinner() tea.Cmd {
	if m.spinnerRunning {
		return nil
	}
	m.spinnerRunning = true
	return spinnerTick()
}

func (m Model) spinner() string {
	return spinnerFrames[m.spinnerFrame%len(spinnerFrames)]
}

const queryElapsedInterval = time.Second

type queryElapsedTickMsg struct {
	session uint64
	request uint64
	elapsed time.Duration
}

func queryElapsedTick(session, request uint64, startedAt time.Time) tea.Cmd {
	return tea.Tick(queryElapsedInterval, func(now time.Time) tea.Msg {
		return queryElapsedTickMsg{
			session: session,
			request: request,
			elapsed: now.Sub(startedAt).Truncate(time.Second),
		}
	})
}
