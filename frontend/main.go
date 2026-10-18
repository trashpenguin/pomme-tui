package main

import (
	"context"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/k1y0miiii/applemusic-tui/engine"
	"github.com/k1y0miiii/applemusic-tui/lastfm"
	"github.com/k1y0miiii/applemusic-tui/lyrics"
	"github.com/k1y0miiii/applemusic-tui/mpris"
	"github.com/k1y0miiii/applemusic-tui/visualizer"
)

const (
	audioInitializingText     = "initializing Apple Music audio…"
	audioInitializingWarning  = "Apple Music is still initializing · R reload"
	visualizerUnavailableNote = "visualizer unavailable · simulated"
	visualizerFrameFreshness  = 400 * time.Millisecond
)

// The live palette. Mutable because themes swap it at runtime; safe because
// Bubble Tea runs Update and View on one goroutine.
var (
	accent    lipgloss.Color
	accentHi  lipgloss.Color
	accentLo  lipgloss.Color
	fgBright  lipgloss.Color
	fgDim     lipgloss.Color
	fgFaint   lipgloss.Color
	borderDim lipgloss.Color
	selBg     lipgloss.Color
)

func init() { applyTheme(themes[0]) }

type phase int

const (
	phaseBoot phase = iota
	phaseReady
	phaseFail
)

type (
	tickMsg   time.Time
	pollMsg   struct{}
	statusMsg string
	noteMsg   string
	readyMsg  struct{ eng *engine.Engine }
	failMsg   struct{ err error }
	stateMsg  struct {
		st  engine.State
		err error
	}
	searchMsg struct {
		category string
		offset   int

		res       engine.SearchResults
		err       error
		keepInput bool // library load keeps the input focused
	}
	discoveryMsg struct {
		feed engine.Discovery
		err  error
	}
	discoveryTracksMsg struct {
		tracks []engine.Track
		err    error
	}
	statsMsg struct {
		res engine.SearchResults
		err error
	}
	lyricsMsg struct {
		id string // track the lyrics belong to
		ly lyrics.Lyrics
	}
	artMsg struct {
		id  string // track the artwork belongs to
		img image.Image
	}
	recentArtMsg struct {
		id  string
		img image.Image
	}
	accountMsg struct {
		acct engine.Account
		err  error
	}
	signedOutMsg        struct{ err error }
	mprisQuitMsg        struct{}
	visualizerOpenedMsg struct {
		service *visualizer.Service
		err     error
		source  string
	}
)

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type model struct {
	helpPage int

	saveQueueUntil time.Time

	libraryCategory  int
	libraryOffset    int
	libraryPaged     bool
	librarySelection int
	mini             bool
	sleepUntil       time.Time
	sleepChoice      int
	searchHistory    []string
	historyIndex     int

	w, h        int
	t           float64 // seconds since start, drives animation
	phase       phase
	status      string
	note        string
	noteAt      float64 // m.t when the note was set; expires after 6s
	eng         *engine.Engine
	connections *connectionLifecycle
	st          engine.State
	focus       int // queue, library, recently played, player
	selIdx      int

	// recently played track history
	recent             []engine.Track
	recentSel          int
	recentArtRequested map[string]bool

	themeName string            // active theme, persisted across runs
	cfg       map[string]string // parsed config.toml, read once at startup

	statusCh chan string

	badPolls       int  // consecutive failed or signed-out state polls
	mprisListening bool // the quit listener is running; a reconnect must not add a second

	// queue-replacement in flight (slow setQueue on playlists/albums)
	loading         string  // title being loaded, "" = none
	loadSnap        string  // state snapshot at fire time; change = done
	loadStart       float64 // m.t at fire time, for the give-up timeout
	loadHideQueue   bool    // true when the queue itself is being replaced
	initStart       float64 // m.t when the DRM audio initialization began
	initFailed      bool    // a MusicKit error ends this initialization attempt
	initPending     bool    // a local play attempt awaits engine confirmation
	initSeen        bool    // a poll observed Initializing during the pending attempt
	initReadyPolls  int     // consecutive same-snapshot Playing polls while pending
	initTimerActive bool    // initStart is valid and the 30s warning timer is running

	// lyrics for the current track
	ly     lyrics.Lyrics
	lyFor  string // track ID the fetch was fired for
	lyBusy bool

	// album artwork for the current track
	art      image.Image
	artFor   string    // track ID the image belongs to
	artCache *artCache // decoded covers, shared across tracks

	// real spectrum; falls back to fake animation
	vizService  *visualizer.Service
	vizOpening  bool
	vizLive     bool
	vizSource   string
	vizTerminal bool
	vizHistory  [48][32]float64
	vizBands    [32]float64 // smoothed, 0..1
	vizTargets  [32]float64
	vizPeaks    [32]float64 // peak-hold markers, decay slowly
	vizMode     int         // vizBars or vizOrb, persisted across runs
	orbSpin     float64     // torus rotation, accelerated by the beat
	orbWobble   float64     // torus tilt oscillation
	orbKick     float64     // 0..1 beat strength, spikes on a bass hit
	orbKickBase float64     // running average the kick is measured against
	wv          wave        // recorded loudness across the current track

	helpOpen bool // the ? overlay; any key dismisses it

	// the a overlay: who is signed in, and the way back out
	acctOpen    bool
	acctLoading bool
	acctBusy    bool // a sign-out is in flight
	acct        engine.Account

	// desktop integration; nil off Linux or without a session bus
	mpris     *mpris.Server
	mprisQuit chan struct{}

	// Last.fm, nil unless configured and authorized
	scrobbler *lastfm.Client
	scrKey    string        // artist+title of the play being timed
	scrStart  time.Time     // wall clock when it started, for the timestamp
	scrPlayed time.Duration // time actually listened, so a seek cannot fake it
	scrDone   bool          // already scrobbled this play

	// search overlay
	searchOpen bool
	sInput     bool
	sQuery     string
	sTab, sSel int
	sBusy      bool
	sRes       engine.SearchResults

	discoveryOpen      bool
	discoveryBusy      bool
	discoveryErr       string
	discoveryFeed      engine.Discovery
	discoverySel       int
	discoveryDetail    bool
	discoveryTracks    []engine.Track
	discoveryTrackSel  int
	discoveryTitle     string
	saveConfirmOpen    bool
	saveConfirmTrack   engine.Track
	queueOpen          bool
	queuePrevFocus     int
	statsOpen          bool
	statsBusy          bool
	statsErr           string
	statsRes           engine.SearchResults
	statsRange         statsRange
	listenHistory      *listeningHistory
	listenLastTrack    string
	listenLastSave     time.Time
	libraryOpen        bool
	libraryErr         string
	libraryDetailTitle string
}

func (m model) visualizerTitle() string {
	if m.vizLive {
		return "VISUALIZER · LIVE · " + m.vizSource
	}
	if m.vizOpening {
		return "VISUALIZER · STARTING"
	}
	return "VISUALIZER · SIMULATED"
}

func (m *model) failVisualizer() {
	m.vizService = nil
	m.vizOpening = false
	m.vizLive = false
	m.vizSource = ""
	if !m.vizTerminal {
		m.vizTerminal = true
		m.note = visualizerUnavailableNote
		m.noteAt = m.t
	}
}

func (m *model) closeVisualizerAsync() {
	service := m.vizService
	m.vizService = nil
	m.vizOpening = false
	m.vizLive = false
	m.vizSource = ""
	closeServiceAsync(service)
}

func closeServiceAsync(service *visualizer.Service) {
	if service != nil {
		go func() {
			_ = service.Close()
		}()
	}
}

func (m model) audioInitializing() bool {
	return (m.st.Initializing || m.initPending) && !m.initFailed
}

func (m *model) beginAudioInitialization() {
	m.initPending = true
	m.initSeen = false
	m.initReadyPolls = 0
	m.initFailed = false
	m.initStart = m.t
	m.initTimerActive = true
}

func (m *model) resetAudioInitialization() {
	m.initPending = false
	m.initSeen = false
	m.initReadyPolls = 0
	m.initFailed = false
	m.initStart = 0
	m.initTimerActive = false
}

func (m *model) failAudioInitialization() {
	m.initPending = false
	m.initSeen = false
	m.initReadyPolls = 0
	m.initFailed = true
	m.initStart = 0
	m.initTimerActive = false
}

func (m model) audioInitializingLine() string {
	sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	return lipgloss.NewStyle().Foreground(accentHi).Render(" "+string(sp[int(m.t*12)%len(sp)])+" ") +
		lipgloss.NewStyle().Foreground(fgDim).Render(audioInitializingText)
}

// badPollsToReconnect is 5 polls ≈ 2.5s — long enough that a busy browser or a
// single 10s eval timeout never costs the user a browser restart.
const badPollsToReconnect = 5

// reconnect drops the dead browser and re-runs startup's Connect, which opens a
// visible login window on its own when the session is really gone.
func (m model) reconnect(status string) (tea.Model, tea.Cmd) {
	eng := m.eng
	m.closeVisualizerAsync()
	m.vizTerminal = false
	m.mpris.Close()
	m.mpris, m.eng = nil, nil
	m.badPolls = 0
	m.st = engine.State{}
	m.phase, m.status = phaseBoot, status
	return m, reconnectCmd(eng, m.statusCh, m.connections)
}

// The old browser must be gone before the new one starts: both use the same
// profile directory, and Chrome holds a lock on it.
func reconnectCmd(eng *engine.Engine, ch chan string, lifecycle ...*connectionLifecycle) tea.Cmd {
	return func() tea.Msg {
		if eng != nil {
			eng.Close()
		}
		return connectCmd(ch, lifecycle...)()
	}
}

func connectCmd(ch chan string, lifecycle ...*connectionLifecycle) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var owner *connectionLifecycle
		if len(lifecycle) > 0 {
			owner = lifecycle[0]
		}
		if owner != nil {
			if !owner.begin() {
				return failMsg{context.Canceled}
			}
			ctx = owner.ctx
		}
		eng, err := engine.ConnectContext(ctx, func(s string) {
			select {
			case ch <- s:
			case <-ctx.Done():
			}
		})
		if owner != nil {
			owner.finish(eng)
		}
		if err != nil {
			return failMsg{err}
		}
		return readyMsg{eng}
	}
}

func listenStatus(ch chan string) tea.Cmd {
	return func() tea.Msg { return statusMsg(<-ch) }
}

func (m model) fetchState() tea.Cmd {
	eng := m.eng
	return func() tea.Msg {
		st, err := eng.State()
		return stateMsg{st, err}
	}
}

func openVisualizerCmd(openSource func() (visualizer.Source, error)) tea.Cmd {
	return func() tea.Msg {
		source, err := openSource()
		if err != nil {
			return visualizerOpenedMsg{err: err}
		}
		sourceName := ""
		if source != nil {
			sourceName = source.Name()
		}
		service, err := visualizer.NewService(source)
		if err != nil {
			if source != nil {
				_ = source.Close()
			}
			return visualizerOpenedMsg{err: err}
		}
		return visualizerOpenedMsg{service: service, source: sourceName}
	}
}

func openNativeVisualizerCmd(eng *engine.Engine) tea.Cmd {
	return func() tea.Msg {
		service, err := visualizer.NewNativeService(eng.Spectrum)
		return visualizerOpenedMsg{service: service, err: err, source: "native playback"}
	}
}

func closeVisualizerCmd(service *visualizer.Service) tea.Cmd {
	return func() tea.Msg {
		_ = service.Close()
		return nil
	}
}

func visualizerFrameStale(now, frameAt time.Time) bool {
	return !now.IsZero() && !frameAt.IsZero() &&
		now.Sub(frameAt) > visualizerFrameFreshness
}

func snap(st engine.State) string {
	return fmt.Sprintf("%s|%d|%d", st.Now.ID, len(st.Queue), st.QueuePos)
}

func doCmd(f func() error) tea.Cmd {
	return func() tea.Msg {
		if err := f(); err != nil {
			return noteMsg(err.Error())
		}
		return nil
	}
}

func fetchLyricsCmd(id, artist, title string, dur time.Duration) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ly, err := lyrics.Fetch(ctx, artist, title, dur)
		if err != nil {
			ly = lyrics.Lyrics{} // network miss = just "no lyrics", not an error banner
		}
		return lyricsMsg{id: id, ly: ly}
	}
}

func fetchArtworkCmd(id, template string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), artworkTimeout)
		defer cancel()
		img, err := fetchArtwork(ctx, artworkURL(template, artworkPx))
		if err != nil {
			return artMsg{id: id} // nil image: the panel just shows lyrics only
		}
		return artMsg{id: id, img: img}
	}
}

const recentArtworkPx = 96
const recentArtworkCols = 6
const recentArtworkRows = 3
const recentCardRows = recentArtworkRows

func fetchRecentArtworkCmd(id, template string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), artworkTimeout)
		defer cancel()
		img, err := fetchArtwork(ctx, artworkURL(template, recentArtworkPx))
		if err != nil {
			return recentArtMsg{id: id}
		}
		return recentArtMsg{id: id, img: img}
	}
}

// requestRecentArtwork fetches only the thumbnails currently visible in the
// compact history panel. The shared bounded cache keeps memory use predictable.
func (m model) requestRecentArtwork() (model, tea.Cmd) {
	if m.searchOpen || m.layout().recH == 0 || len(m.recent) == 0 {
		return m, nil
	}
	start, count := m.recentWindow(m.layout().recH - 3)
	if m.recentArtRequested == nil {
		m.recentArtRequested = make(map[string]bool)
	}
	var cmds []tea.Cmd
	for i := start; i < start+count; i++ {
		track := m.recent[i]
		if track.Art == "" || m.recentArtRequested[track.ID] {
			continue
		}
		if _, ok := m.artCache.get(track.ID); ok {
			m.recentArtRequested[track.ID] = false
			continue
		}
		m.recentArtRequested[track.ID] = true
		cmds = append(cmds, fetchRecentArtworkCmd(track.ID, track.Art))
	}
	return m, tea.Batch(cmds...)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.SetWindowTitle("Pomme"), tick(), connectCmd(m.statusCh, m.connections), listenStatus(m.statusCh))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		if m.focus == focusLibrary && !m.libraryVisible() {
			m.focus = focusQueue
		}
		if m.focus == focusRecent && !m.recentVisible() {
			if m.libraryVisible() {
				m.focus = focusLibrary
			} else {
				m.focus = focusPlayer
			}
		}
	case tickMsg:
		if !m.sleepUntil.IsZero() && !time.Now().Before(m.sleepUntil) {
			m.sleepUntil = time.Time{}
			m.sleepChoice = 0
			m.note, m.noteAt = "Sleep timer finished", m.t
			if m.eng != nil {
				return m, tea.Batch(tick(), doCmd(m.eng.Pause))
			}
		}
		var visualizerClose tea.Cmd
		now := time.Time(msg)
		m.t += 1.0 / 30
		if m.phase == phaseReady && m.st.Playing && m.st.Now.ID != "" {
			if m.listenHistory == nil {
				m.listenHistory = &listeningHistory{Days: make(map[string]map[string]listeningRecord)}
			}
			newPlay := m.listenLastTrack != m.st.Now.ID
			m.listenLastTrack = m.st.Now.ID
			m.listenHistory.record(m.st.Now, now, time.Second/30, newPlay)
			if m.listenLastSave.IsZero() || now.Sub(m.listenLastSave) >= 15*time.Second {
				m.listenHistory.save()
				m.listenLastSave = now
			}
		} else if m.st.Now.ID == "" {
			m.listenLastTrack = ""
		}
		if m.phase == phaseReady && m.st.Playing && m.st.Pos < m.st.Dur {
			m.st.Pos += time.Second / 30 // optimistic between polls
		}
		if m.loading != "" && m.t-m.loadStart > 12 && !m.audioInitializing() {
			m.loading = "" // give up quietly
		}
		if m.audioInitializing() && m.initTimerActive &&
			m.t-m.initStart > 30 && m.note != audioInitializingWarning {
			m.note, m.noteAt = audioInitializingWarning, m.t
		}
		if m.note != "" && m.t-m.noteAt > 6 &&
			!(m.audioInitializing() && m.note == audioInitializingWarning) {
			m.note = ""
		}
		if m.vizService != nil {
			service := m.vizService
			if service.Err() != nil {
				m.failVisualizer()
				visualizerClose = closeVisualizerCmd(service)
			} else if frame, ok := service.Latest(); ok {
				if visualizerFrameStale(now, frame.At) {
					m.vizTargets = [32]float64{}
				} else {
					m.vizTargets = frame.Bands
					m.vizLive = frame.Live
					m.vizSource = frame.Source
					m.vizOpening = false
				}
			}
		}
		if m.vizLive {
			for i := range m.vizBands {
				m.vizBands[i] += (m.vizTargets[i] - m.vizBands[i]) * 0.45
			}
		} else if !m.vizOpening {
			m.vizBands = simulatedBands(m.t, m.st.Playing)
		} else {
			m.vizBands = [32]float64{}
		}
		copy(m.vizHistory[1:], m.vizHistory[:47])
		m.vizHistory[0] = m.vizBands
		decayPeaks(&m.vizPeaks, m.vizBands)
		m.orbKickBase, m.orbKick = bassKick(m.orbKickBase, m.orbKick, bassLevel(m.vizBands))
		m.orbSpin, m.orbWobble = orbAdvance(m.orbSpin, m.orbWobble, m.orbKick)
		if m.st.Dur > 0 && m.st.Playing {
			m.wv.record(float64(m.st.Pos)/float64(m.st.Dur), bandsLevel(m.vizBands))
		}
		scrobble := m.advanceScrobble(time.Second / 30)
		cmds := []tea.Cmd{tick()}
		if visualizerClose != nil {
			cmds = append(cmds, visualizerClose)
		}
		if scrobble != nil {
			cmds = append(cmds, scrobble)
		}
		return m, tea.Batch(cmds...)
	case statusMsg:
		m.status = string(msg)
		return m, listenStatus(m.statusCh)
	case readyMsg:
		m.phase, m.eng = phaseReady, msg.eng
		cmds := []tea.Cmd{
			m.fetchState(),
			m.libraryCmd(), // fills the recently-played grid
		}
		// System audio capture outlives the browser, so a reconnect keeps the
		// visualizer it already has instead of opening a second source.
		if m.vizService == nil {
			m.vizOpening = true
			cmds = append(cmds, openNativeVisualizerCmd(m.eng))
		}
		// A machine with no session bus — a TTY, a container, macOS — simply
		// has no MPRIS, which is not an error worth showing. MPRIS is re-published
		// after a reconnect because its controls hold the old engine.
		if srv, err := mpris.Publish(mprisControls{eng: m.eng, quit: m.mprisQuit}); err == nil && srv != nil {
			m.mpris = srv
			if !m.mprisListening {
				m.mprisListening = true
				cmds = append(cmds, listenMPRISQuit(m.mprisQuit))
			}
		} else if err != nil && runtime.GOOS == "windows" {
			m.note, m.noteAt = "Global shortcuts disabled: "+err.Error(), m.t
		}
		return m, tea.Batch(cmds...)
	case mprisQuitMsg:
		m.closeVisualizerAsync()
		m.mpris.Close()
		m.listenHistory.save()
		if m.eng != nil {
			m.eng.Close()
		}
		return m, tea.Quit
	case visualizerOpenedMsg:
		if m.phase != phaseReady || !m.vizOpening {
			if msg.service != nil {
				return m, closeVisualizerCmd(msg.service)
			}
			return m, nil
		}
		if msg.err != nil || msg.service == nil {
			m.failVisualizer()
			return m, nil
		}
		m.vizService = msg.service
		m.vizOpening = false
		m.vizLive = true
		m.vizSource = msg.source
	case failMsg:
		m.phase, m.status = phaseFail, msg.err.Error()
	case pollMsg:
		if m.eng == nil {
			return m, nil // a reconnect is in flight; readyMsg restarts the loop
		}
		return m, m.fetchState()
	case stateMsg:
		// A dead session shows up two ways: isAuthorized flips false, or — once
		// the page has navigated away from MusicKit — the poll fails outright.
		// Either way it must persist, so a single hiccup cannot restart us.
		if msg.err != nil || !msg.st.Authed {
			m.badPolls++
			if m.badPolls >= badPollsToReconnect {
				return m.reconnect("session lost — signing in to Apple Music again…")
			}
		} else {
			m.badPolls = 0
		}
		var cmd tea.Cmd
		if msg.err == nil {
			wasInitializing := m.st.Initializing
			newSt := msg.st
			// keep the optimistic clock for small drift: the polled position
			// lags by up to the poll interval and would flick lyrics back
			if newSt.Now.ID == m.st.Now.ID {
				if d := newSt.Pos - m.st.Pos; d > -1500*time.Millisecond && d < 1500*time.Millisecond {
					newSt.Pos = m.st.Pos
				}
			}
			m.st = newSt
			m.mpris.Update(mprisState(m.st))
			nowInitializing := m.st.Initializing
			if m.initPending && nowInitializing {
				m.initSeen = true
				m.initReadyPolls = 0
			}
			if nowInitializing && !m.initTimerActive {
				m.initStart = m.t
				m.initTimerActive = true
			}
			if m.initPending && !nowInitializing {
				complete := false
				if m.st.Playing {
					complete = m.initSeen || snap(m.st) != m.loadSnap
					if !complete {
						m.initReadyPolls++
						complete = m.initReadyPolls >= 2
					}
				} else {
					m.initReadyPolls = 0
				}
				if complete {
					m.resetAudioInitialization()
					m.loading = ""
				}
			}
			m.selIdx = min(m.selIdx, max(0, len(m.st.Queue)-1))
			if m.st.Err != "" {
				m.note, m.noteAt = m.st.Err, m.t // MusicKit rejection from an action
				m.failAudioInitialization()
				m.loading = ""
			} else if wasInitializing && !nowInitializing {
				m.initFailed = false
				if !m.initPending {
					m.initStart = 0
					m.initTimerActive = false
				}
			}
			if m.loading != "" && snap(m.st) != m.loadSnap && !m.audioInitializing() {
				m.loading = "" // queue switched — loading done
			}
			if id := m.st.Now.ID; id != "" && id != m.lyFor {
				m.lyFor, m.lyBusy, m.ly = id, true, lyrics.Lyrics{}
				cmd = fetchLyricsCmd(id, m.st.Now.Artist, m.st.Now.Title, m.st.Now.Duration)
			}
			if id := m.st.Now.ID; id != "" && id != m.artFor {
				m.wv.reset()
				m.artFor, m.art = id, nil
				if cached, ok := m.artCache.get(id); ok {
					m.art = cached
					m.applyAutoTheme()
				} else if tmpl := m.st.Now.Art; tmpl != "" {
					cmd = tea.Batch(cmd, fetchArtworkCmd(id, tmpl))
				}
			}
		}
		poll := tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return pollMsg{} })
		return m, tea.Batch(poll, cmd)
	case lyricsMsg:
		if msg.id == m.lyFor {
			m.ly, m.lyBusy = msg.ly, false
		}
	case artMsg:
		if msg.id == m.artFor && msg.img != nil {
			m.art = msg.img
			m.artCache.put(msg.id, msg.img)
			m.applyAutoTheme()
		}
	case recentArtMsg:
		if msg.img != nil {
			m.artCache.put(msg.id, msg.img)
			if m.recentArtRequested != nil {
				m.recentArtRequested[msg.id] = false
			}
		} else if m.recentArtRequested != nil {
			m.recentArtRequested[msg.id] = true // don't retry a known missing cover on every scroll
		}
	case discoveryMsg:
		m.discoveryBusy = false
		m.discoveryErr = ""
		if msg.err != nil {
			m.discoveryErr = msg.err.Error()
		} else {
			m.discoveryFeed = msg.feed
			m.discoverySel = min(m.discoverySel, max(0, m.discoveryItemCount()-1))
		}
	case discoveryTracksMsg:
		m.discoveryBusy = false
		if msg.err != nil {
			m.discoveryErr = msg.err.Error()
			m.note, m.noteAt = msg.err.Error(), m.t
		} else {
			m.discoveryErr = ""
			m.discoveryTracks = msg.tracks
			m.discoveryTrackSel = 0
			m.discoveryDetail = true
		}
	case statsMsg:
		m.statsBusy = false
		if msg.err != nil {
			m.statsErr = msg.err.Error()
		} else {
			m.statsRes = msg.res
			m.statsErr = ""
		}
	case noteMsg:
		m.note, m.noteAt = string(msg), m.t
		m.loading = "" // action failed — stop the spinner
		if m.st.Initializing || m.initPending {
			m.failAudioInitialization()
		}
	case searchMsg:
		m.libraryPaged = msg.category != ""
		if m.libraryOpen {
			m.libraryErr = ""
			if msg.err != nil {
				m.libraryErr = msg.err.Error()
			}
		}
		if msg.category != "" {
			m.sTab = map[string]int{"songs": 1, "favorites": 1, "albums": 2, "playlists": 3, "artists": 4}[msg.category]
			m.note, m.noteAt = fmt.Sprintf("Library %s | page %d | PgUp/PgDn", msg.category, msg.offset/50+1), m.t
		}
		m.sBusy = false
		if msg.err != nil {
			m.note, m.noteAt = msg.err.Error(), m.t
		}
		// Apple grants the library and the listening history as separate
		// privileges, so one can be refused while the other answers — whatever
		// did arrive is shown alongside the reason rather than dropped with it.
		m.sRes, m.sSel, m.sInput = msg.res, 0, msg.keepInput
		if !msg.keepInput && m.sTab == 0 {
			m.sTab = 1 // catalog search has no RECENT — land on SONGS
		}
		// Only a library load carries recently played; a catalog search must
		// not wipe the grid.
		if len(msg.res.Recent) > 0 {
			m.recent = msg.res.Recent
			m.recentSel = min(m.recentSel, len(m.recent)-1)
		}
	case accountMsg:
		m.acctLoading = false
		if msg.err == nil {
			m.acct = msg.acct
		}
		// A failed lookup is not worth a banner: the overlay still shows
		// "signed in" and the one thing it is there for still works.
	case signedOutMsg:
		m.acctOpen, m.acctBusy, m.acct = false, false, engine.Account{}
		if msg.err != nil {
			m.note, m.noteAt = "sign out failed: "+msg.err.Error(), m.t
			return m, nil
		}
		return m.reconnect("signed out — sign in to Apple Music")
	case tea.MouseMsg:
		if m.mini {
			return m, nil
		}
		return m.updateMouse(msg)
	case tea.KeyMsg:
		// Any key dismisses help, and that key does nothing else — reading the
		// list should never fire the command you were looking up.
		if m.helpOpen {
			if pageDirection(msg) > 0 || msg.String() == "right" {
				m.helpPage = min(m.helpPage+1, m.helpPageCount()-1)
				return m, nil
			}
			if pageDirection(msg) < 0 || msg.String() == "left" {
				m.helpPage = max(0, m.helpPage-1)
				return m, nil
			}
			m.helpOpen = false
			return m, nil
		}
		if m.acctOpen {
			return m.updateAccount(msg)
		}
		if m.libraryOpen {
			return m.updateLibraryView(msg)
		}
		if m.queueOpen {
			return m.updateQueueManager(msg)
		}
		if m.discoveryOpen {
			return m.updateDiscovery(msg)
		}
		if m.statsOpen {
			return m.updateStats(msg)
		}
		if m.searchOpen {
			return m.updateSearch(msg)
		}
		return m.updateKeys(msg)
	}
	return m, nil
}

// Focus zones, cycled by Tab in visual order down the left column and then to
// the transport bar.
const (
	focusQueue = iota
	focusLibrary
	focusRecent
	focusPlayer
)

func (m model) libraryVisible() bool { return m.layout().libH > 0 }

func (m model) recentVisible() bool { return m.layout().recH > 0 }

func (m model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.phase != phaseReady {
		if msg.String() == "R" && m.phase == phaseFail {
			return m.reconnect("retrying connection...")
		}
		if s := msg.String(); s == "q" || s == "ctrl+c" {
			m.closeVisualizerAsync()
			return m, tea.Quit
		}
		return m, nil
	}
	if m.audioInitializing() {
		switch msg.String() {
		case " ", "n", "p", "enter":
			m.note, m.noteAt = audioInitializingText, m.t
			return m, nil
		}
	}
	eng := m.eng
	if direction := pageDirection(msg); direction != 0 {
		switch m.focus {
		case focusQueue:
			m.selIdx = max(0, min(m.selIdx+direction*max(1, m.layout().queueRows()), len(m.st.Queue)-1))
		case focusLibrary:
			m.librarySelection = max(0, min(m.librarySelection+direction*4, 6))
		case focusRecent:
			m.recentSel = max(0, min(m.recentSel+direction*4, len(m.recent)-1))
		}
		if m.focus == focusRecent {
			var cmd tea.Cmd
			m, cmd = m.requestRecentArtwork()
			return m, cmd
		}
		return m, nil
	}
	switch msg.String() {
	case "q", "ctrl+c":
		m.closeVisualizerAsync()
		m.mpris.Close()
		m.listenHistory.save()
		if eng != nil {
			eng.Close()
		}
		return m, tea.Quit
	case "tab":
		m.focus = (m.focus + 1) % 4
		if m.focus == focusLibrary && !m.libraryVisible() {
			m.focus = focusPlayer
		} else if m.focus == focusRecent && !m.recentVisible() {
			m.focus = focusPlayer
		}
	case " ":
		m.st.Playing = !m.st.Playing
		return m, doCmd(eng.PlayPause)
	case "n":
		return m, doCmd(eng.Next)
	case "p":
		return m, doCmd(eng.Prev)
	case "s":
		return m, doCmd(eng.ToggleShuffle)
	case "r":
		return m, doCmd(eng.CycleRepeat)
	case "?":
		m.helpPage = 0
		m.helpOpen = true
	case "a":
		m.acctOpen, m.acctLoading = true, true
		return m, accountCmd(eng)
	case "S":
		if time.Now().After(m.saveQueueUntil) {
			m.saveQueueUntil = time.Now().Add(5 * time.Second)
			m.note, m.noteAt = "Press S again within 5s to save queue to Apple Music", m.t
			return m, nil
		}
		m.saveQueueUntil = time.Time{}
		name := "Pomme Queue " + time.Now().Format("2006-01-02 15:04")
		return m, func() tea.Msg {
			if err := eng.SavePlaylist(name); err != nil {
				return noteMsg(err.Error())
			}
			return noteMsg("Saved playlist: " + name)
		}
	case "L":
		selection := 1 // Open Songs when Library is not focused.
		if m.focus == focusLibrary {
			selection = m.librarySelection
		}
		return m.openLibrarySelection(selection)
	case "delete":
		if m.focus == focusQueue {
			index := m.selIdx
			return m, doCmd(func() error { return eng.EditQueue("remove", index, 0) })
		}
	case "ctrl+up", "ctrl+down":
		if m.focus == focusQueue {
			index := m.selIdx
			target := index - 1
			if msg.String() == "ctrl+down" {
				target = index + 1
			}
			if target >= 0 && target < len(m.st.Queue) {
				m.selIdx = target
				return m, doCmd(func() error { return eng.EditQueue("move", index, target) })
			}
		}
	case "ctrl+delete":
		return m, doCmd(func() error { return eng.EditQueue("clear", 0, 0) })
	case "m":
		m.mini = !m.mini
	case "T":
		choices := []int{0, 15, 30, 60, 90}
		m.sleepChoice = (m.sleepChoice + 1) % len(choices)
		m.sleepUntil = time.Time{}
		m.note = "Sleep timer off"
		if minutes := choices[m.sleepChoice]; minutes > 0 {
			m.sleepUntil = time.Now().Add(time.Duration(minutes) * time.Minute)
			m.note = fmt.Sprintf("Sleep in %d minutes", minutes)
		}
		m.noteAt = m.t
	case "v":
		m.vizMode = (m.vizMode + 1) % vizModes
		saveVizMode(m.vizMode)
		m.note, m.noteAt = "visualizer · "+vizModeName(m.vizMode), m.t
	case "t":
		next := nextTheme(m.themeName)
		m.themeName = next.name
		applyTheme(next)
		saveThemeName(next.name)
		m.applyAutoTheme()
		m.note, m.noteAt = "theme · "+next.name, m.t
	case "R": // reload the web player when it wedges
		m.note, m.noteAt = "reloading engine…", m.t
		m.lyFor = ""
		m.resetAudioInitialization()
		return m, doCmd(eng.Reload)
	case "/":
		m.searchOpen, m.sInput = true, true
		if strings.TrimSpace(m.sQuery) == "" {
			m.sBusy, m.sTab, m.sSel = true, 0, 0 // open on recently played
			return m, m.libraryCmd()
		}
	case "j", "down":
		switch m.focus {
		case focusQueue:
			m.selIdx = min(m.selIdx+1, max(0, len(m.st.Queue)-1))
		case focusLibrary:
			m.librarySelection = min(m.librarySelection+1, 6)
		case focusRecent:
			m.recentSel = min(m.recentSel+1, max(0, len(m.recent)-1))
		default:
			m.st.Volume = max(m.st.Volume-5, 0)
			v := m.st.Volume
			return m, doCmd(func() error { return eng.SetVolume(v) })
		}
		if m.focus == focusRecent {
			return m.requestRecentArtwork()
		}
	case "k", "up":
		switch m.focus {
		case focusQueue:
			m.selIdx = max(m.selIdx-1, 0)
		case focusLibrary:
			m.librarySelection = max(m.librarySelection-1, 0)
		case focusRecent:
			m.recentSel = max(m.recentSel-1, 0)
		default:
			m.st.Volume = min(m.st.Volume+5, 100)
			v := m.st.Volume
			return m, doCmd(func() error { return eng.SetVolume(v) })
		}
		if m.focus == focusRecent {
			return m.requestRecentArtwork()
		}
	case "enter":
		if m.focus == focusQueue && len(m.st.Queue) > 0 {
			i := m.selIdx
			m.loading = m.st.Queue[i].Title
			m.loadSnap, m.loadStart, m.loadHideQueue = snap(m.st), m.t, false
			return m, doCmd(func() error { return eng.JumpTo(i) })
		}
		if m.focus == focusRecent {
			if m.recentSel < len(m.recent) {
				tr := m.recent[m.recentSel]
				m.loading, m.loadSnap, m.loadStart = tr.Title, snap(m.st), m.t
				m.loadHideQueue = true
				m.beginAudioInitialization()
				kind, id := tr.Kind, tr.ID
				play := doCmd(func() error { return eng.Play(kind, id) })
				m, art := m.requestRecentArtwork()
				return m, tea.Batch(play, art)
			}
		}
		if m.focus == focusLibrary {
			return m.openLibrarySelection(m.librarySelection)
		}
	case "left":
		if m.focus == focusLibrary {
			m.librarySelection = max(m.librarySelection-1, 0)
			return m, nil
		}
		if m.focus == focusPlayer {
			m.st.Pos = max(m.st.Pos-5*time.Second, 0)
			m.wv.reset()
			p := m.st.Pos
			return m, doCmd(func() error { return eng.SeekTo(p) })
		}
	case "right":
		if m.focus == focusLibrary {
			m.librarySelection = min(m.librarySelection+1, 6)
			return m, nil
		}
		if m.focus == focusPlayer {
			m.st.Pos = min(m.st.Pos+5*time.Second, m.st.Dur)
			m.wv.reset()
			p := m.st.Pos
			return m, doCmd(func() error { return eng.SeekTo(p) })
		}
	}
	return m, nil
}

// pad truncates/pads a styled line to exactly w columns.
func pad(s string, w int) string {
	if lipgloss.Width(s) > w {
		s = lipgloss.NewStyle().MaxWidth(w).Render(s)
	}
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}

func centeredRows(s string, w, rows int) string {
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := ""
		if r == rows/2 {
			line = strings.Repeat(" ", max(0, (w-lipgloss.Width(s))/2)) + s
		}
		b.WriteString(pad(line, w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// panel draws a bordered box. accentNow is the (possibly bass-pulsed) accent
// used for the focused border; callers pass m.pulsedAccent().
func panel(title, body string, w, h int, focused bool, accentNow lipgloss.Color) string {
	bc, tc := borderDim, fgFaint
	if focused {
		bc, tc = accentNow, accentHi
	}
	head := lipgloss.NewStyle().Foreground(tc).Bold(true).Render(" " + title)
	content := pad(head, w) + "\n" + body
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).BorderForeground(bc).
		Width(w).Height(h).Render(content)
}

func (m model) queuePanel(w, h int) string {
	var b strings.Builder
	rows := h - 1
	if m.audioInitializing() {
		return centeredRows(m.audioInitializingLine(), w, rows)
	}
	if m.loading != "" && m.loadHideQueue {
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		line := lipgloss.NewStyle().Foreground(accentHi).Render(" "+string(sp[int(m.t*12)%len(sp)])+" ") +
			lipgloss.NewStyle().Foreground(fgDim).Render("loading ") +
			lipgloss.NewStyle().Foreground(fgBright).Bold(true).Render(m.loading) +
			lipgloss.NewStyle().Foreground(fgDim).Render("…")
		for r := 0; r < rows; r++ {
			s := ""
			if r == rows/2 {
				s = line
			}
			b.WriteString(pad(s, w) + "\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
	q := m.st.Queue
	if len(q) == 0 {
		empty := lipgloss.NewStyle().Foreground(fgFaint).Render(" queue is empty — press / to search")
		return pad(empty, w)
	}
	start := max(0, min(m.selIdx-rows/2, len(q)-rows))
	for r := 0; r < rows; r++ {
		i := start + r
		if i >= len(q) {
			b.WriteString(pad("", w) + "\n")
			continue
		}
		tr := q[i]
		prefix, ts, as := "  ", fgDim, fgFaint
		if i == m.st.QueuePos {
			prefix, ts, as = "▶ ", fgBright, fgDim
		}
		left := lipgloss.NewStyle().Foreground(accentHi).Render(prefix) +
			lipgloss.NewStyle().Foreground(ts).Render(tr.Title) +
			lipgloss.NewStyle().Foreground(as).Render(" — "+tr.Artist)
		dur := lipgloss.NewStyle().Foreground(fgFaint).Render(fmtTime(tr.Duration))
		gap := w - lipgloss.Width(left) - lipgloss.Width(dur) - 1
		line := left + strings.Repeat(" ", max(1, gap)) + dur
		if m.focus == focusQueue && i == m.selIdx {
			line = lipgloss.NewStyle().Background(selBg).Render(pad(line, w))
		}
		b.WriteString(pad(line, w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// recentTitle keeps the selected track's position visible while the list scrolls.
func (m model) recentTitle() string {
	if len(m.recent) == 0 {
		return "RECENTLY PLAYED"
	}
	return fmt.Sprintf("RECENTLY PLAYED · %d/%d", m.recentSel+1, len(m.recent))
}

// recentWindow maps the visible body height to whole artwork cards, keeping
// the selected track visible as the user moves through listening history.
func (m model) recentWindow(rows int) (start, count int) {
	visible := rows / recentCardRows
	if visible <= 0 || len(m.recent) == 0 {
		return 0, 0
	}
	count = min(visible, len(m.recent))
	start = max(0, min(m.recentSel-count/2, len(m.recent)-count))
	return start, count
}

// recentListPanel draws a compact, scrollable list beneath the library box.
// The containing panel uses one row for its title, leaving h-1 rows for tracks.
func (m model) recentListPanel(w, h int) string {
	rows := max(0, h-1)
	if rows == 0 {
		return ""
	}
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	if len(m.recent) == 0 {
		return centeredRows(faint.Render(" nothing played yet"), w, rows)
	}
	start, count := m.recentWindow(rows)
	var lines []string
	for visible := 0; visible < count; visible++ {
		i := start + visible
		track := m.recent[i]
		style, mark := faint, "  "
		if m.focus == focusRecent && i == m.recentSel {
			style, mark = lipgloss.NewStyle().Foreground(accentHi).Bold(true), "› "
		}
		thumb := make([]string, recentArtworkRows)
		for r := range thumb {
			thumb[r] = lipgloss.NewStyle().Foreground(borderDim).Render("░░░░░░")
		}
		if img, ok := m.artCache.get(track.ID); ok && img != nil {
			if cover := renderArtwork(img, recentArtworkCols, recentArtworkRows); len(cover) == recentArtworkRows {
				thumb = cover
			}
		}
		label := mark + track.Title
		lines = append(lines, pad(thumb[0]+"  "+style.Render(label), w))
		lines = append(lines, pad(thumb[1]+"  "+faint.Render(track.Artist), w))
		lines = append(lines, pad(thumb[2]+"  "+faint.Render(" "), w))
	}
	for len(lines) < rows {
		lines = append(lines, pad("", w))
	}
	return strings.Join(lines, "\n")
}

func liveBarHeights(bands [32]float64, bars, rows int) []float64 {
	if bars <= 0 {
		return nil
	}
	heights := make([]float64, bars)
	for i := range heights {
		start := i * len(bands) / bars
		end := (i + 1) * len(bands) / bars
		if end <= start {
			end = start + 1
		}
		end = min(end, len(bands))

		var peak float64
		for _, value := range bands[start:end] {
			peak = max(peak, value)
		}
		heights[i] = min(peak*1.15, 1) * float64(max(rows, 0))
	}
	return heights
}

// peakDecay is the fall rate of the hold markers, per 30 fps frame.
const peakDecay = 0.012

// decayPeaks makes each peak follow its band up instantly and fall slowly —
// the Winamp behaviour.
func decayPeaks(peaks *[32]float64, bands [32]float64) {
	for i := range peaks {
		if bands[i] > peaks[i] {
			peaks[i] = bands[i]
			continue
		}
		peaks[i] = max(peaks[i]-peakDecay, 0)
	}
}

// waveBuckets is the resolution of the recorded waveform. Wider than any
// realistic progress bar, so a resize never loses detail.
const waveBuckets = 240

// wave records the audio level at each point of the track as it plays, so the
// progress bar can show the shape of what was already heard.
type wave struct {
	samples [waveBuckets]float64
}

// record stores level at position frac (0..1), keeping the loudest value seen
// in that bucket.
func (w *wave) record(frac, level float64) {
	i := int(frac * waveBuckets)
	i = min(max(i, 0), waveBuckets-1)
	if level > w.samples[i] {
		w.samples[i] = level
	}
}

func (w *wave) reset() { w.samples = [waveBuckets]float64{} }

// column returns the peak level for column col of a bar cols wide.
func (w *wave) column(col, cols int) float64 {
	if cols <= 0 {
		return 0
	}
	start := col * waveBuckets / cols
	end := (col + 1) * waveBuckets / cols
	end = min(max(end, start+1), waveBuckets)
	peak := 0.0
	for _, v := range w.samples[start:end] {
		peak = max(peak, v)
	}
	return peak
}

// bandsLevel is the overall loudness: the mean of all bands.
func bandsLevel(bands [32]float64) float64 {
	sum := 0.0
	for _, v := range bands {
		sum += v
	}
	return sum / float64(len(bands))
}

// bassLevel is the mean of the four lowest bands — what drives the pulse.
func bassLevel(bands [32]float64) float64 {
	sum := 0.0
	for _, v := range bands[:4] {
		sum += v
	}
	return sum / 4
}

// pulseAmount is the current bass energy, or 0 when the pulse is switched off.
func (m model) pulseAmount() float64 {
	if !configBool(m.cfg, "visualizer.pulse", true) || !m.st.Playing {
		return 0
	}
	return bassLevel(m.vizBands)
}

func (m model) pulsedAccent() lipgloss.Color { return pulse(accent, m.pulseAmount()) }

// simulatedBands is the fallback animation, moved out of vizPanel so the peak
// markers and the waveform see the same values in live and simulated modes.
func simulatedBands(t float64, playing bool) [32]float64 {
	var bands [32]float64
	for i := range bands {
		x := t*1.35 + float64(i)*0.55
		a := 0.24 + 0.62*math.Abs(math.Sin(x))*(0.5+0.5*math.Sin(t*0.43+float64(i)*1.9))
		if !playing {
			a *= 0.06 // ponytail: bars just collapse on pause, no decay animation
		}
		bands[i] = min(max(a, 0), 1)
	}
	return bands
}

func (m model) vizPanel(w, h int) string {
	switch m.vizMode {
	case vizWave, vizMirror, vizWaterfall:
		return spectrumPanel(w, h-1, m.vizMode, m.vizBands, m.vizHistory)
	case vizTorus:
		return orbPanel(w, h-1, m.orbSpin, m.orbWobble, m.vizBands)
	case vizSphere:
		return spherePanel(w, h-1, m.orbSpin, m.orbWobble, m.orbKick, m.vizBands)
	}
	rows, bars := h-1, max(1, w/3)
	heights := liveBarHeights(m.vizBands, bars, rows)
	var peaks []float64
	if configBool(m.cfg, "visualizer.peaks", true) {
		peaks = liveBarHeights(m.vizPeaks, bars, rows)
	}
	partial := []rune("▁▂▃▄▅▆▇█")
	var b strings.Builder
	for r := 0; r < rows; r++ {
		level := float64(rows - r)
		zone := level / float64(rows)
		color := accentLo
		if zone > 0.66 {
			color = accentHi
		} else if zone > 0.33 {
			color = accent
		}
		st := lipgloss.NewStyle().Foreground(color)
		markSt := lipgloss.NewStyle().Foreground(fgDim)
		var row strings.Builder
		for i := 0; i < bars; i++ {
			// A marker only shows where the bar itself is not already drawn.
			mark := peaks != nil && peaks[i] > level-1 && peaks[i] <= level &&
				heights[i] <= level-1
			switch {
			case heights[i] >= level:
				row.WriteString(st.Render("██"))
			case heights[i] > level-1:
				c := string(partial[min(7, int((heights[i]-level+1)*8))])
				row.WriteString(st.Render(c + c))
			case mark:
				row.WriteString(markSt.Render("▔▔"))
			default:
				row.WriteString("  ")
			}
			row.WriteString(" ")
		}
		b.WriteString(pad(" "+row.String(), w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) lyricsPanel(w, h int) string {
	rows := h - 1
	artCols, artRows := 0, 0
	if configBool(m.cfg, "artwork.enabled", true) && m.art != nil {
		artCols, artRows = artworkLayout(w, rows, configInt(m.cfg, "artwork.min_lyrics_width", 30), m.lyricsWidth())
	}
	textW := w
	if artCols > 0 {
		textW = w - artCols - artworkGap
	}
	text := m.lyricsText(textW, rows)
	if artCols == 0 {
		return text
	}
	cover := renderArtwork(m.art, artCols, artRows)
	lines := strings.Split(text, "\n")
	top := (rows - artRows) / 2
	var b strings.Builder
	for r := 0; r < rows; r++ {
		line := ""
		if r < len(lines) {
			line = lines[r]
		}
		// Center the cover vertically against the lyrics block.
		left := strings.Repeat(" ", artCols)
		if r >= top && r-top < len(cover) {
			left = cover[r-top]
		}
		b.WriteString(left + strings.Repeat(" ", artworkGap) + pad(line, textW) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// lyricsWidth is how many columns the lyrics actually use: the widest line plus
// the one-column indent lyricsText draws. Zero when there are no lines, which
// leaves the cover bounded by the configured minimum instead.
func (m model) lyricsWidth() int {
	n := 0
	for _, l := range m.ly.Lines {
		if w := lipgloss.Width(l.Text); w > n {
			n = w
		}
	}
	if n == 0 {
		return 0
	}
	return n + 1
}

// lyricsText is the text column of the lyrics panel, sized independently of the
// cover so the two can sit side by side.
func (m model) lyricsText(w, rows int) string {
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	center := func(s string) string {
		var b strings.Builder
		for r := 0; r < rows; r++ {
			line := ""
			if r == rows/2 {
				line = s
			}
			b.WriteString(pad(line, w) + "\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
	switch {
	case m.lyBusy:
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		return center(faint.Render(" " + string(sp[int(m.t*12)%len(sp)]) + " looking up lyrics…"))
	case len(m.ly.Lines) == 0:
		return center(faint.Render(" no lyrics for this track"))
	}
	// pick the line to center on: by timestamp when synced, by progress when plain
	cur := -1
	if m.ly.Synced {
		cur = m.ly.Current(m.st.Pos)
	} else if m.st.Dur > 0 {
		cur = int(float64(m.st.Pos) / float64(m.st.Dur) * float64(len(m.ly.Lines)))
	}
	start := max(0, min(cur-rows/2, len(m.ly.Lines)-rows))
	var b strings.Builder
	for r := 0; r < rows; r++ {
		i := start + r
		if i >= len(m.ly.Lines) {
			b.WriteString(pad("", w) + "\n")
			continue
		}
		text := m.ly.Lines[i].Text
		st := faint // past lines
		if i > cur {
			st = lipgloss.NewStyle().Foreground(fgDim) // upcoming, slightly brighter
		}
		if m.ly.Synced && i == cur {
			st = lipgloss.NewStyle().Foreground(accentHi).Bold(true)
			if text == "" {
				text = "♪"
			}
		}
		b.WriteString(pad(st.Render(" "+text), w) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func fmtTime(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// fmtLongTime is fmtTime with an hours field for queue totals.
func fmtLongTime(d time.Duration) string {
	s := int(d.Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, (s%3600)/60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// queueTitle labels the queue panel with the position and the time left,
// counting the unplayed part of the current track plus everything after it.
func (m model) queueTitle() string {
	q := m.st.Queue
	if len(q) == 0 || m.st.QueuePos < 0 || m.st.QueuePos >= len(q) {
		return "QUEUE"
	}
	left := q[m.st.QueuePos].Duration - m.st.Pos
	if left < 0 {
		left = 0
	}
	for _, tr := range q[m.st.QueuePos+1:] {
		left += tr.Duration
	}
	return fmt.Sprintf("QUEUE · %d/%d · %s left",
		m.st.QueuePos+1, len(q), fmtLongTime(left))
}

func (m model) transportPanel(w int) string {
	icon := "="
	if !m.st.Playing {
		icon = ">"
	}
	dim := lipgloss.NewStyle().Foreground(fgDim)
	faint := lipgloss.NewStyle().Foreground(fgFaint)
	pink := lipgloss.NewStyle().Foreground(accentHi)

	title, artist := m.st.Now.Title, m.st.Now.Artist
	if title == "" {
		title, artist = "nothing playing", "press / to find music"
	}
	hints := transportHints(m.focus)
	if m.note != "" {
		hints = m.note + " "
	}
	l1 := pink.Render(" ▀▀ ") +
		lipgloss.NewStyle().Foreground(fgBright).Bold(true).Render(title) +
		faint.Render(" — "+artist)
	if album := m.st.Now.Album; album != "" {
		l1 += faint.Render(" · " + album)
	}
	if m.audioInitializing() {
		l1 = m.audioInitializingLine()
	} else if m.loading != "" {
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		l1 = pink.Render(" "+string(sp[int(m.t*12)%len(sp)])+" ") +
			dim.Render("loading ") +
			lipgloss.NewStyle().Foreground(fgBright).Bold(true).Render(m.loading) +
			dim.Render("…")
	}
	// The hints share this row with the track title, so on a narrow terminal
	// there may be no room for them. pad() truncates silently, which is how the
	// old line lost its tail without anyone noticing; drop to the short form
	// instead, and then to nothing, so whatever is shown is shown whole.
	if !m.sleepUntil.IsZero() {
		l1 += dim.Render(" | sleep " + time.Until(m.sleepUntil).Round(time.Second).String())
	}
	hints = fitHints(hints, w-lipgloss.Width(l1)-1)
	r1 := faint.Render(hints)
	line1 := l1 + strings.Repeat(" ", max(1, w-lipgloss.Width(l1)-lipgloss.Width(r1))) + r1

	volSegs := (m.st.Volume + 9) / 17 // 0..6 segments
	volSt := faint
	if m.focus == focusPlayer {
		volSt = pink
	}
	shuf, rep := "off", [3]string{"off", "one", "all"}[min(m.st.Repeat, 2)]
	if m.st.Shuffle {
		shuf = "on"
	}
	r2 := faint.Render(fmt.Sprintf("⇄ %s  ↻ %s  vol ", shuf, rep)) +
		volSt.Render(strings.Repeat("▮", volSegs)+strings.Repeat("▯", 6-volSegs)) +
		dim.Render(fmt.Sprintf(" %d%% ", m.st.Volume))
	times := fmtTime(m.st.Pos) + " "
	timee := " " + fmtTime(m.st.Dur)
	_, barW := m.progressGeometry(w)
	frac := 0.0
	if m.st.Dur > 0 {
		frac = min(float64(m.st.Pos)/float64(m.st.Dur), 1)
	}
	filled := int(frac * float64(barW))
	bar := m.progressBar(barW, filled)
	l2 := lipgloss.NewStyle().Foreground(accent).Render(" ▄▄ ") +
		pink.Render(icon+" ") + dim.Render(times) + bar + dim.Render(timee)
	line2 := l2 + strings.Repeat(" ", max(1, w-lipgloss.Width(l2)-lipgloss.Width(r2))) + r2
	return pad(line1, w) + "\n" + pad(line2, w)
}

// progressBar draws the played part of the track as the recorded waveform and
// the rest as a flat rule. Falls back to a plain filled bar when the waveform
// is disabled in the config.
func (m model) progressBar(barW, filled int) string {
	barW = max(0, barW)
	filled = max(0, min(filled, barW))
	rest := lipgloss.NewStyle().Foreground(borderDim).
		Render(strings.Repeat("─", max(0, barW-filled)))
	if !configBool(m.cfg, "visualizer.wave", false) {
		return lipgloss.NewStyle().Foreground(pulse(accentHi, m.pulseAmount())).
			Render(strings.Repeat("━", filled)) + rest
	}
	steps := []rune("▁▂▃▄▅▆▇█")
	st := lipgloss.NewStyle().Foreground(pulse(accentHi, m.pulseAmount()))
	var b strings.Builder
	for c := 0; c < filled; c++ {
		lvl := m.wv.column(c, barW)
		b.WriteString(st.Render(string(steps[min(int(lvl*float64(len(steps))), len(steps)-1)])))
	}
	return b.String() + rest
}

func (m model) bootView() string {
	logo := lipgloss.NewStyle().Foreground(accentHi).Bold(true).Render("[ >_ ] pomme")
	sub := lipgloss.NewStyle().Foreground(fgFaint).Render("apple music in your terminal")
	var line string
	if m.phase == phaseFail {
		line = lipgloss.NewStyle().Foreground(accent).Render("✕ "+m.status) +
			lipgloss.NewStyle().Foreground(fgFaint).Render("  ·  R retry · q quit")
	} else {
		sp := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		line = lipgloss.NewStyle().Foreground(fgDim).
			Render(string(sp[int(m.t*12)%len(sp)]) + " " + m.status)
	}
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
		logo+"\n"+sub+"\n\n"+line)
}

func (m model) View() string {
	if m.mini && m.phase == phaseReady && !m.searchOpen && !m.helpOpen && !m.acctOpen {
		return m.miniView()
	}
	if m.w < 70 || m.h < 22 {
		return "\n  terminal too small — need at least 70×22\n"
	}
	if m.phase != phaseReady {
		return m.bootView()
	}
	if m.helpOpen {
		return m.helpView()
	}
	if m.acctOpen {
		return m.accountView()
	}
	if m.libraryOpen {
		return m.libraryView()
	}
	if m.queueOpen {
		return m.queueManagerView()
	}
	if m.discoveryOpen {
		return m.discoveryView()
	}
	if m.statsOpen {
		return m.statsView()
	}
	if m.searchOpen {
		return m.searchView()
	}
	searchBar := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderDim).
		Width(m.w - 2).Render(lipgloss.NewStyle().Foreground(fgFaint).Render(" Search  ") +
		lipgloss.NewStyle().Foreground(fgBright).Render(m.sQuery) +
		lipgloss.NewStyle().Foreground(fgFaint).Render("   press / to search"))
	lay := m.layout()

	left := panel("pomme · "+m.queueTitle(), m.queuePanel(lay.qw, lay.qh), lay.qw, lay.qh,
		m.focus == focusQueue, m.pulsedAccent())
	if lay.libH > 0 {
		library := []string{"Discovery", "Songs", "Albums", "Playlists", "Artists", "Favorites", "Stats"}
		var items []string
		for i, name := range library {
			mark := "  "
			if i == m.librarySelection {
				mark = "› "
			}
			items = append(items, mark+name)
		}
		left = lipgloss.JoinVertical(lipgloss.Left, left,
			panel("LIBRARY", strings.Join(items[:min(len(items), max(1, lay.libH-3))], "\n"), lay.qw, lay.libH-2,
				m.focus == focusLibrary, m.pulsedAccent()))
	}
	if lay.recH > 0 {
		left = lipgloss.JoinVertical(lipgloss.Left, left,
			panel(m.recentTitle(), m.recentListPanel(lay.qw, lay.recH-2), lay.qw, lay.recH-2,
				m.focus == focusRecent, m.pulsedAccent()))
	}
	viz := panel(m.visualizerTitle(), m.vizPanel(lay.vw, lay.vh-2), lay.vw, lay.vh-2,
		false, m.pulsedAccent())
	lyr := panel("LYRICS", m.lyricsPanel(lay.vw, lay.lh), lay.vw, lay.lh, false, m.pulsedAccent())
	right := lipgloss.JoinVertical(lipgloss.Left, viz, lyr)
	top := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	tw := m.w - 2
	transport := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(map[bool]lipgloss.Color{true: m.pulsedAccent(), false: borderDim}[m.focus == focusPlayer]).
		Width(tw).Render(m.transportPanel(tw))

	return lipgloss.JoinVertical(lipgloss.Left, searchBar, top, transport)
}

// version is stamped at build time with -ldflags "-X main.version=...".
// Release builds carry the tag; a plain `go build` stays "dev".
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("Pomme TUI %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
			return
		case "lastfm-auth":
			os.Exit(runLastfmAuth())
		}
	}

	// Direct launches must run the same dependency setup as Pomme.exe.
	if runtime.GOOS == "windows" && os.Getenv("AMTUI_NATIVE_BACKEND") == "" {
		executable, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		script := filepath.Join(filepath.Dir(executable), "Run-Pomme.ps1")
		if _, err := os.Stat(script); err == nil {
			command := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script)
			command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := command.Run(); err != nil {
				log.Fatal(err)
			}
			return
		}
	}
	connections := newConnectionLifecycle()
	defer connections.shutdown()
	m := model{
		connections: connections,
		statusCh:    make(chan string, 8),
		status:      "starting…",
		mprisQuit:   make(chan struct{}, 1),
	}
	m.searchHistory = loadSearchHistory()
	m.listenHistory = loadListeningHistory()
	m.historyIndex = -1
	m.cfg = loadConfig()
	m.vizMode = loadVizMode()
	m.scrobbler = newScrobbler(m.cfg)
	m.artCache = newArtCache(8)
	t := themeFromConfig(m.cfg, loadThemeName())
	m.themeName = t.name
	applyTheme(t)
	// Cell motion rather than all motion: the app only reacts to clicks and the
	// wheel, and all-motion floods the loop with a message per pointer move.
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	m.listenHistory.save()
	connections.shutdown()
	if err != nil {
		log.Fatal(err)
	}
}
