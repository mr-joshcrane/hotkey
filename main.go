package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image/color"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// Pattern holds a pattern with optional friendly name
type Pattern struct {
	Name    string
	Pattern string
}

// Default patterns (used if no file found)
var defaultPatterns = []Pattern{
	{"5 Group Cycle", "1a2a3a4a5a"},
	{"4 Group Cycle", "1a2a3a4a"},
	{"3 Group Cycle", "1a2a3a"},
	{"F-Key Cycle", "F1aF2aF3a"},
	{"Click Practice", "LCaRCa"},
}

// patternsFile is the config file name
const patternsFile = "keystroke_patterns.txt"

// Key mappings for special keys
var keyNames = map[fyne.KeyName]string{
	fyne.KeyF1:     "F1",
	fyne.KeyF2:     "F2",
	fyne.KeyF3:     "F3",
	fyne.KeyF4:     "F4",
	fyne.KeyF5:     "F5",
	fyne.KeyF6:     "F6",
	fyne.KeyF7:     "F7",
	fyne.KeyF8:     "F8",
	fyne.KeyF9:     "F9",
	fyne.KeyF10:    "F10",
	fyne.KeyF11:    "F11",
	fyne.KeyF12:    "F12",
	fyne.KeyReturn: "Enter",
	fyne.KeyEnter:  "Enter",
	fyne.KeySpace:  " ",
	fyne.KeyEscape: "ESC",
}

// Display icons for special inputs
var displayIcons = map[string]string{
	"LC":  "◐",
	"RC":  "◑",
	"MC":  "◉",
	"SLC": "⇧◐",
	"SRC": "⇧◑",
	"F1":  "[F1]",
	"F2":  "[F2]",
	"F3":  "[F3]",
	"F4":  "[F4]",
	"F5":  "[F5]",
	"F6":  "[F6]",
	"F7":  "[F7]",
	"F8":  "[F8]",
	"F9":  "[F9]",
	"F10": "[F10]",
	"F11": "[F11]",
	"F12": "[F12]",
}

// formatForDisplay converts pattern codes to visual icons
func formatForDisplay(s string) string {
	result := s
	for _, token := range []string{"SLC", "SRC", "F10", "F11", "F12", "LC", "RC", "MC", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9"} {
		if icon, ok := displayIcons[token]; ok {
			result = strings.ReplaceAll(result, token, icon)
		}
	}
	return result
}

// getExpectedKey extracts the key token at a given character position in a pattern
func getExpectedKey(pattern string, charPos int) string {
	tokens := []string{"SLC", "SRC", "F10", "F11", "F12", "LC", "RC", "MC", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9"}
	pos := 0
	for pos < len(pattern) {
		found := false
		for _, token := range tokens {
			if strings.HasPrefix(pattern[pos:], token) {
				if pos == charPos {
					return token
				}
				pos += len(token)
				found = true
				break
			}
		}
		if !found {
			if pos == charPos {
				return string(pattern[pos])
			}
			pos++
		}
	}
	return "?"
}

// loadPatterns loads patterns from the config file, or returns defaults
func loadPatterns() []Pattern {
	patterns, err := loadPatternsFromFile(patternsFile)
	if err == nil && len(patterns) > 0 {
		return patterns
	}

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		patterns, err = loadPatternsFromFile(filepath.Join(exeDir, patternsFile))
		if err == nil && len(patterns) > 0 {
			return patterns
		}
	}

	return defaultPatterns
}

func loadPatternsFromFile(path string) ([]Pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []Pattern
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if parts := strings.SplitN(line, "|", 2); len(parts) == 2 {
			patterns = append(patterns, Pattern{Name: parts[0], Pattern: parts[1]})
		} else {
			patterns = append(patterns, Pattern{Name: line, Pattern: line})
		}
	}

	return patterns, scanner.Err()
}

// Statistics types
const statsFile = "keystroke_stats.json"

type Mistake struct {
	Position  int       `json:"position"`
	Expected  string    `json:"expected"`
	Actual    string    `json:"actual"`
	Timestamp time.Time `json:"timestamp"`
}

type PatternStats struct {
	Pattern       string        `json:"pattern"`
	Name          string        `json:"name"`
	TotalAttempts int           `json:"total_attempts"`
	PerfectCount  int           `json:"perfect_count"`
	TotalResets   int           `json:"total_resets"`
	BestTime      time.Duration `json:"best_time"`
	TotalTime     time.Duration `json:"total_time"`
	CurrentStreak int           `json:"current_streak"`
	BestStreak    int           `json:"best_streak"`
	LastPracticed time.Time     `json:"last_practiced"`
	Mistakes      []Mistake     `json:"mistakes"`
}

type SessionRecord struct {
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	Duration        time.Duration `json:"duration"`
	PatternsTotal   int           `json:"patterns_total"`
	PatternsPerfect int           `json:"patterns_perfect"`
	Completed       bool          `json:"completed"`
}

type AllStats struct {
	PatternStats   map[string]*PatternStats `json:"pattern_stats"`
	Sessions       []SessionRecord          `json:"sessions"`
	TotalSessions  int                      `json:"total_sessions"`
	TotalTrainTime time.Duration            `json:"total_train_time"`
	LastUpdated    time.Time                `json:"last_updated"`
}

func loadStats() *AllStats {
	stats := &AllStats{
		PatternStats: make(map[string]*PatternStats),
		Sessions:     []SessionRecord{},
	}

	data, err := os.ReadFile(statsFile)
	if err != nil {
		return stats
	}

	json.Unmarshal(data, stats)
	if stats.PatternStats == nil {
		stats.PatternStats = make(map[string]*PatternStats)
	}
	return stats
}

func (s *AllStats) save() error {
	s.LastUpdated = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statsFile, data, 0644)
}

func (s *AllStats) getPatternStats(pattern Pattern) *PatternStats {
	if ps, ok := s.PatternStats[pattern.Pattern]; ok {
		return ps
	}
	ps := &PatternStats{
		Pattern: pattern.Pattern,
		Name:    pattern.Name,
	}
	s.PatternStats[pattern.Pattern] = ps
	return ps
}

func (s *AllStats) recordAttempt(pattern Pattern, elapsed time.Duration, resets int) {
	ps := s.getPatternStats(pattern)
	ps.TotalAttempts++
	ps.TotalTime += elapsed
	ps.TotalResets += resets
	ps.LastPracticed = time.Now()

	if resets == 0 {
		ps.PerfectCount++
		ps.CurrentStreak++
		if ps.CurrentStreak > ps.BestStreak {
			ps.BestStreak = ps.CurrentStreak
		}
		if ps.BestTime == 0 || elapsed < ps.BestTime {
			ps.BestTime = elapsed
		}
	} else {
		ps.CurrentStreak = 0
	}
}

func (s *AllStats) recordMistake(pattern Pattern, position int, expected, actual string) {
	ps := s.getPatternStats(pattern)
	mistake := Mistake{
		Position:  position,
		Expected:  expected,
		Actual:    actual,
		Timestamp: time.Now(),
	}
	ps.Mistakes = append(ps.Mistakes, mistake)
	if len(ps.Mistakes) > 100 {
		ps.Mistakes = ps.Mistakes[len(ps.Mistakes)-100:]
	}
}

func (s *AllStats) startSession() time.Time {
	return time.Now()
}

func (s *AllStats) endSession(startTime time.Time, total, perfect int, completed bool) {
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	session := SessionRecord{
		StartTime:       startTime,
		EndTime:         endTime,
		Duration:        duration,
		PatternsTotal:   total,
		PatternsPerfect: perfect,
		Completed:       completed,
	}
	s.Sessions = append(s.Sessions, session)
	s.TotalSessions++
	s.TotalTrainTime += duration
	s.save()
}

// App holds the application state
type App struct {
	window fyne.Window

	// UI elements
	patternName   *canvas.Text
	targetDisplay *canvas.Text
	inputDisplay  *canvas.Text
	statusLabel   *canvas.Text
	bestTimeLabel *canvas.Text
	progressLabel *canvas.Text
	hintLabel     *canvas.Text

	// Main container that captures input
	mainContainer *FullWindowInput

	// All loaded patterns
	allPatterns  []Pattern
	patternQueue []Pattern
	currentIndex int

	currentPattern Pattern
	inputBuffer    []string
	isActive       bool
	inSession      bool
	startTime      time.Time
	resetCount     int

	// Session stats
	sessionPerfect int
	sessionTotal   int
	sessionStart   time.Time

	// Persistent stats
	stats *AllStats
}

// FullWindowInput captures all input for the entire window
type FullWindowInput struct {
	widget.BaseWidget
	app        *App
	focused    bool
	background *canvas.Rectangle
	content    fyne.CanvasObject
}

func NewFullWindowInput(app *App, content fyne.CanvasObject) *FullWindowInput {
	fw := &FullWindowInput{
		app:        app,
		background: canvas.NewRectangle(color.RGBA{25, 25, 35, 255}),
		content:    content,
	}
	fw.ExtendBaseWidget(fw)
	return fw
}

func (fw *FullWindowInput) CreateRenderer() fyne.WidgetRenderer {
	return &fullWindowRenderer{
		fw:         fw,
		background: fw.background,
		content:    fw.content,
	}
}

type fullWindowRenderer struct {
	fw         *FullWindowInput
	background *canvas.Rectangle
	content    fyne.CanvasObject
}

func (r *fullWindowRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	r.content.Resize(size)
}

func (r *fullWindowRenderer) MinSize() fyne.Size {
	return r.content.MinSize()
}

func (r *fullWindowRenderer) Refresh() {
	r.background.Refresh()
	r.content.Refresh()
}

func (r *fullWindowRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background, r.content}
}

func (r *fullWindowRenderer) Destroy() {}

// Focusable interface
func (fw *FullWindowInput) FocusGained() {
	fw.focused = true
}

func (fw *FullWindowInput) FocusLost() {
	if fw.app.inSession {
		go func() {
			time.Sleep(10 * time.Millisecond)
			fyne.Do(func() {
				fw.app.window.Canvas().Focus(fw)
			})
		}()
		return
	}
	fw.focused = false
}

func (fw *FullWindowInput) Focused() bool {
	return fw.focused
}

// Tappable interface
func (fw *FullWindowInput) Tapped(e *fyne.PointEvent) {
	fw.app.window.Canvas().Focus(fw)
}

// TypedKey handles special keys
func (fw *FullWindowInput) TypedKey(key *fyne.KeyEvent) {
	// ESC stops the session
	if key.Name == fyne.KeyEscape && fw.app.inSession {
		fw.app.stopSession()
		return
	}

	// Space or Enter starts session when not active
	if !fw.app.inSession && (key.Name == fyne.KeySpace || key.Name == fyne.KeyReturn || key.Name == fyne.KeyEnter) {
		fw.app.startSession()
		return
	}

	if !fw.app.isActive {
		return
	}

	// Map special keys
	if name, ok := keyNames[key.Name]; ok {
		if name != "ESC" && name != "Enter" {
			fw.app.addKey(name)
		}
	}
}

// TypedRune handles regular character input
func (fw *FullWindowInput) TypedRune(r rune) {
	if !fw.app.isActive {
		return
	}
	fw.app.addKey(string(r))
}

// MouseDown handles mouse clicks
var _ desktop.Mouseable = (*FullWindowInput)(nil)

func (fw *FullWindowInput) MouseDown(e *desktop.MouseEvent) {
	fw.app.window.Canvas().Focus(fw)

	if !fw.app.isActive {
		return
	}

	shift := e.Modifier&fyne.KeyModifierShift != 0

	switch e.Button {
	case desktop.MouseButtonPrimary:
		if shift {
			fw.app.addKey("SLC")
		} else {
			fw.app.addKey("LC")
		}
	case desktop.MouseButtonSecondary:
		if shift {
			fw.app.addKey("SRC")
		} else {
			fw.app.addKey("RC")
		}
	case desktop.MouseButtonTertiary:
		fw.app.addKey("MC")
	}
}

func (fw *FullWindowInput) MouseUp(e *desktop.MouseEvent) {}

func main() {
	a := app.NewWithID("com.buildorder.keystroketrainer")
	w := a.NewWindow("⌨️ Keystroke Trainer")
	w.Resize(fyne.NewSize(700, 450))

	myApp := &App{
		window:      w,
		allPatterns: loadPatterns(),
		stats:       loadStats(),
	}

	myApp.setupUI()
	w.ShowAndRun()
}

func (app *App) setupUI() {
	// Pattern name - large and prominent
	app.patternName = canvas.NewText("", color.RGBA{100, 180, 255, 255})
	app.patternName.TextSize = 28
	app.patternName.TextStyle = fyne.TextStyle{Bold: true}
	app.patternName.Alignment = fyne.TextAlignCenter

	// Best time motivation
	app.bestTimeLabel = canvas.NewText("", color.RGBA{150, 150, 150, 255})
	app.bestTimeLabel.TextSize = 16
	app.bestTimeLabel.Alignment = fyne.TextAlignCenter

	// Target display - THE MAIN FOCUS
	app.targetDisplay = canvas.NewText("", color.RGBA{80, 220, 120, 255})
	app.targetDisplay.TextSize = 56
	app.targetDisplay.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
	app.targetDisplay.Alignment = fyne.TextAlignCenter

	// Input display - what user has typed
	app.inputDisplay = canvas.NewText("", color.RGBA{200, 200, 200, 255})
	app.inputDisplay.TextSize = 56
	app.inputDisplay.TextStyle = fyne.TextStyle{Monospace: true}
	app.inputDisplay.Alignment = fyne.TextAlignCenter

	// Status feedback
	app.statusLabel = canvas.NewText("", color.RGBA{255, 255, 255, 255})
	app.statusLabel.TextSize = 24
	app.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	app.statusLabel.Alignment = fyne.TextAlignCenter

	// Progress
	app.progressLabel = canvas.NewText("", color.RGBA{150, 150, 180, 255})
	app.progressLabel.TextSize = 18
	app.progressLabel.Alignment = fyne.TextAlignCenter

	// Hint at bottom
	app.hintLabel = canvas.NewText("Press SPACE to start • ESC to stop", color.RGBA{80, 80, 100, 255})
	app.hintLabel.TextSize = 14
	app.hintLabel.Alignment = fyne.TextAlignCenter

	// Initial state
	app.showIdleState()

	// Build the layout - centered, minimal
	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(app.patternName),
		container.NewCenter(app.bestTimeLabel),
		layout.NewSpacer(),
		container.NewCenter(app.targetDisplay),
		container.NewPadded(container.NewCenter(app.inputDisplay)),
		layout.NewSpacer(),
		container.NewCenter(app.statusLabel),
		container.NewCenter(app.progressLabel),
		layout.NewSpacer(),
		container.NewCenter(app.hintLabel),
	)

	// Wrap in full-window input capture
	app.mainContainer = NewFullWindowInput(app, container.NewPadded(content))
	app.window.SetContent(app.mainContainer)

	// Auto-focus on show
	app.window.Canvas().Focus(app.mainContainer)
}

func (app *App) showIdleState() {
	app.patternName.Text = "⌨️ Keystroke Trainer"
	app.patternName.Color = color.RGBA{100, 180, 255, 255}
	app.patternName.Refresh()

	app.bestTimeLabel.Text = fmt.Sprintf("%d patterns loaded", len(app.allPatterns))
	app.bestTimeLabel.Refresh()

	app.targetDisplay.Text = ""
	app.targetDisplay.Refresh()

	app.inputDisplay.Text = ""
	app.inputDisplay.Refresh()

	app.statusLabel.Text = "Click anywhere to focus"
	app.statusLabel.Color = color.RGBA{150, 150, 150, 255}
	app.statusLabel.Refresh()

	app.progressLabel.Text = ""
	app.progressLabel.Refresh()

	app.hintLabel.Text = "Press SPACE to start • ESC to stop"
	app.hintLabel.Refresh()
}

func (app *App) shufflePatterns() {
	app.patternQueue = make([]Pattern, len(app.allPatterns))
	copy(app.patternQueue, app.allPatterns)

	rand.Shuffle(len(app.patternQueue), func(i, j int) {
		app.patternQueue[i], app.patternQueue[j] = app.patternQueue[j], app.patternQueue[i]
	})
}

func (app *App) startSession() {
	app.shufflePatterns()
	app.currentIndex = 0
	app.inSession = true
	app.sessionPerfect = 0
	app.sessionTotal = 0
	app.sessionStart = app.stats.startSession()

	app.hintLabel.Text = "ESC to stop session"
	app.hintLabel.Refresh()

	app.window.Canvas().Focus(app.mainContainer)
	app.nextPattern()
}

func (app *App) stopSession() {
	app.inSession = false
	app.isActive = false

	app.stats.endSession(app.sessionStart, app.sessionTotal, app.sessionPerfect, false)

	app.statusLabel.Text = fmt.Sprintf("Session ended: %d/%d perfect", app.sessionPerfect, app.sessionTotal)
	app.statusLabel.Color = color.RGBA{200, 200, 100, 255}
	app.statusLabel.Refresh()

	app.progressLabel.Text = ""
	app.progressLabel.Refresh()

	app.patternName.Text = "Session Stopped"
	app.patternName.Color = color.RGBA{200, 150, 100, 255}
	app.patternName.Refresh()

	app.bestTimeLabel.Text = ""
	app.bestTimeLabel.Refresh()

	app.targetDisplay.Text = ""
	app.targetDisplay.Refresh()

	app.inputDisplay.Text = ""
	app.inputDisplay.Refresh()

	app.hintLabel.Text = "Press SPACE to start new session"
	app.hintLabel.Refresh()
}

func (app *App) sessionComplete() {
	app.inSession = false
	app.isActive = false

	app.stats.endSession(app.sessionStart, app.sessionTotal, app.sessionPerfect, true)

	elapsed := time.Since(app.sessionStart)

	app.patternName.Text = "🏆 ALL PATTERNS MASTERED!"
	app.patternName.Color = color.RGBA{255, 215, 0, 255}
	app.patternName.Refresh()

	app.bestTimeLabel.Text = fmt.Sprintf("Session time: %v", elapsed.Round(time.Second))
	app.bestTimeLabel.Refresh()

	app.targetDisplay.Text = "🎉"
	app.targetDisplay.Refresh()

	app.inputDisplay.Text = ""
	app.inputDisplay.Refresh()

	app.statusLabel.Text = fmt.Sprintf("%d patterns completed perfectly", len(app.allPatterns))
	app.statusLabel.Color = color.RGBA{100, 255, 100, 255}
	app.statusLabel.Refresh()

	app.progressLabel.Text = ""
	app.progressLabel.Refresh()

	app.hintLabel.Text = "Press SPACE to train again"
	app.hintLabel.Refresh()
}

func (app *App) nextPattern() {
	if !app.inSession {
		return
	}

	if len(app.patternQueue) == 0 {
		app.sessionComplete()
		return
	}

	app.currentPattern = app.patternQueue[0]
	app.patternQueue = app.patternQueue[1:]

	app.inputBuffer = []string{}
	app.resetCount = 0
	app.isActive = true
	app.startTime = time.Time{}

	// Update displays
	app.patternName.Text = app.currentPattern.Name
	app.patternName.Color = color.RGBA{100, 180, 255, 255}
	app.patternName.Refresh()

	// Show best time if exists
	if ps, ok := app.stats.PatternStats[app.currentPattern.Pattern]; ok && ps.BestTime > 0 {
		app.bestTimeLabel.Text = fmt.Sprintf("Best: %v", ps.BestTime.Round(time.Millisecond))
		app.bestTimeLabel.Color = color.RGBA{255, 215, 0, 255}
	} else {
		app.bestTimeLabel.Text = "No record yet"
		app.bestTimeLabel.Color = color.RGBA{100, 100, 100, 255}
	}
	app.bestTimeLabel.Refresh()

	app.targetDisplay.Text = formatForDisplay(app.currentPattern.Pattern)
	app.targetDisplay.Color = color.RGBA{80, 220, 120, 255}
	app.targetDisplay.Refresh()

	app.inputDisplay.Text = "▌"
	app.inputDisplay.Color = color.RGBA{150, 150, 150, 255}
	app.inputDisplay.Refresh()

	app.statusLabel.Text = ""
	app.statusLabel.Refresh()

	app.progressLabel.Text = fmt.Sprintf("%d patterns remaining", len(app.patternQueue)+1)
	app.progressLabel.Refresh()

	app.window.Canvas().Focus(app.mainContainer)
}

func (app *App) addKey(key string) {
	if !app.isActive {
		return
	}

	testInput := strings.Join(append(app.inputBuffer, key), "")
	if !strings.HasPrefix(app.currentPattern.Pattern, testInput) {
		if len(app.inputBuffer) == 0 {
			return // Ignore wrong first keystroke
		}

		position := len(testInput) - len(key)
		expected := getExpectedKey(app.currentPattern.Pattern, position)

		app.stats.recordMistake(app.currentPattern, position, expected, key)
		app.stats.save()

		app.resetCount++
		app.inputBuffer = []string{}
		app.statusLabel.Text = fmt.Sprintf("❌ Expected %s", formatForDisplay(expected))
		app.statusLabel.Color = color.RGBA{255, 100, 100, 255}
		app.statusLabel.Refresh()

		app.inputDisplay.Text = "▌"
		app.inputDisplay.Color = color.RGBA{255, 100, 100, 255}
		app.inputDisplay.Refresh()
		return
	}

	// Start timer on first valid keystroke
	if app.startTime.IsZero() {
		app.startTime = time.Now()
	}

	app.inputBuffer = append(app.inputBuffer, key)
	app.updateInputDisplay()

	// Check for completion
	currentInput := strings.Join(app.inputBuffer, "")
	if len(currentInput) >= len(app.currentPattern.Pattern) {
		app.finishPattern()
	}
}

func (app *App) updateInputDisplay() {
	input := strings.Join(app.inputBuffer, "")
	if len(input) == 0 {
		app.inputDisplay.Text = "▌"
		app.inputDisplay.Color = color.RGBA{150, 150, 150, 255}
	} else {
		app.inputDisplay.Text = formatForDisplay(input)
		app.inputDisplay.Color = color.RGBA{100, 255, 100, 255}
	}
	app.inputDisplay.Refresh()
}

func (app *App) finishPattern() {
	if !app.isActive {
		return
	}

	app.isActive = false
	elapsed := time.Since(app.startTime)

	app.sessionTotal++

	// Record stats
	app.stats.recordAttempt(app.currentPattern, elapsed, app.resetCount)
	app.stats.save()

	if app.resetCount == 0 {
		app.sessionPerfect++

		// Check if new best
		ps := app.stats.PatternStats[app.currentPattern.Pattern]
		if elapsed == ps.BestTime {
			app.statusLabel.Text = fmt.Sprintf("✅ NEW BEST! %v", elapsed.Round(time.Millisecond))
			app.statusLabel.Color = color.RGBA{255, 215, 0, 255}
		} else {
			app.statusLabel.Text = fmt.Sprintf("✅ %v", elapsed.Round(time.Millisecond))
			app.statusLabel.Color = color.RGBA{100, 255, 100, 255}
		}
		app.inputDisplay.Color = color.RGBA{0, 255, 0, 255}
	} else {
		app.patternQueue = append(app.patternQueue, app.currentPattern)
		app.statusLabel.Text = fmt.Sprintf("↻ %d resets - retry later", app.resetCount)
		app.statusLabel.Color = color.RGBA{255, 180, 100, 255}
		app.inputDisplay.Color = color.RGBA{255, 200, 100, 255}
	}
	app.statusLabel.Refresh()
	app.inputDisplay.Refresh()

	go func() {
		time.Sleep(400 * time.Millisecond)
		fyne.Do(func() {
			if app.inSession {
				app.nextPattern()
			}
		})
	}()
}
