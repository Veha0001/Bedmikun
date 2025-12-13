package cmd

import (
	"fmt"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NOTE: https://github.com/charmbracelet/bubbletea/blob/main/examples/progress-download/
var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

type tea_dl_model struct {
	pw       *progressWriter
	progress progress.Model
	err      error
}

var p *tea.Program

type progressMsg float64
type progressWriter struct {
	total      int
	downloaded int
	file       *os.File
	reader     io.Reader
	onProgress func(float64)
}
type progressErrMsg struct{ err error }

func finalPause() tea.Cmd {
	return tea.Tick(time.Millisecond*750, func(_ time.Time) tea.Msg {
		return nil
	})
}
func (pw *progressWriter) Start() {
	// TeeReader calls pw.Write() each time a new response is received
	_, err := io.Copy(pw.file, io.TeeReader(pw.reader, pw))
	if err != nil {
		p.Send(progressErrMsg{err})
	}
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.downloaded += len(p)
	if pw.total > 0 && pw.onProgress != nil {
		pw.onProgress(float64(pw.downloaded) / float64(pw.total))
	}
	return len(p), nil
}

func getResponse(url string) (*http.Response, error) {
	resp, err := http.Get(url) // nolint:gosec
	if err != nil {
		logger.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("receiving status of %d for url: %s", resp.StatusCode, url)
	}
	return resp, nil
}
func (m tea_dl_model) Init() tea.Cmd {
	return nil
}

func (m tea_dl_model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 2*2 - 4
		if m.progress.Width > min(80) {
			m.progress.Width = 80
		}
		return m, nil

	case progressErrMsg:
		m.err = msg.err
		return m, tea.Quit

	case progressMsg:
		var cmds []tea.Cmd

		if msg >= 1.0 {
			cmds = append(cmds, tea.Sequence(finalPause(), tea.Quit))
		}

		cmds = append(cmds, m.progress.SetPercent(float64(msg)))
		return m, tea.Batch(cmds...)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m tea_dl_model) View() string {
	if m.err != nil {
		return "Error downloading: " + m.err.Error() + "\n"
	}

	pad := strings.Repeat(" ", 2)
	return "\n" +
		pad + m.progress.View() + "\n\n" +
		pad + helpStyle("Press any key to quit")
}
func GetDownload(url string, dest string) {
	if url == "" {
		logger.Fatal("URL: Must not be empty.")
		os.Exit(1)
	}
	resp, err := getResponse(url)
	if err != nil {
		logger.Error("could not get response", err)
		os.Exit(1)
	}
	defer resp.Body.Close() // nolint:errcheck
	if resp.ContentLength <= 0 {
		fmt.Println("can't parse content length, aborting download")
		os.Exit(1)
	}

	filename := filepath.Base(url)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("could not create file:", err)
		os.Exit(1)
	}
	defer file.Close() // nolint:errcheck

	pw := &progressWriter{
		total:  int(resp.ContentLength),
		file:   file,
		reader: resp.Body,
		onProgress: func(ratio float64) {
			p.Send(progressMsg(ratio))
		},
	}
	m := tea_dl_model{
		pw:       pw,
		progress: progress.New(progress.WithDefaultGradient()),
	}
	p = tea.NewProgram(m)
	go pw.Start()

	if _, err := p.Run(); err != nil {
		fmt.Println("error running program:", err)
		os.Exit(1)
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}
