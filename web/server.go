// Package web serves the PBX Elite pairing website.
//
// Flow (single session):
//   1. Nobody linked yet  → "/" shows the pair page. The owner types the
//      WhatsApp number, gets an 8-character pair code, enters it in
//      WhatsApp → Linked devices → "Link with phone number".
//   2. As soon as pairing succeeds the userbot goes live on that number and
//      "/" switches to the dashboard page.
//   3. Once a session exists the pairing endpoint is closed — only ONE
//      session is ever allowed.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

//go:embed pair.html dash.html
var pages embed.FS

// Group is one block of the command list shown on the dashboard.
type Group struct {
	Icon  string   `json:"icon"`
	Title string   `json:"title"`
	Cmds  []string `json:"cmds"`
}

// Info is the live state of the bot, as JSON for the pages.
type Info struct {
	Paired    bool    `json:"paired"`
	Connected bool    `json:"connected"`
	Number    string  `json:"number"`
	Name      string  `json:"name"`
	BotName   string  `json:"botName"`
	Version   string  `json:"version"`
	Prefix    string  `json:"prefix"`
	Uptime    string  `json:"uptime"`
	UptimeSec int64   `json:"uptimeSec"`
	Commands  int     `json:"commands"`
	NeedKey   bool    `json:"needKey"`
	Groups    []Group `json:"groups,omitempty"`
}

// Hooks connects the web layer to the bot (keeps this package free of imports from modules/).
type Hooks struct {
	Info       func() Info
	Pair       func(number string) (code string, err error)
	GenSession func(number string) (jobID string, code string, err error)
	// Logout unlinks the WhatsApp device and clears the stored session.
	// The process exits after responding so Heroku restarts to the pair page.
	Logout func() error
}

// ─── session-gen job map ─────────────────────────────────────────────────────

// SessionJob holds the state of an in-progress session-string generation.
type SessionJob struct {
	Status  string // "pending" | "code" | "done" | "error"
	Code    string // pair code (available once status=="code")
	Session string // base64 session string (available once status=="done")
	ErrMsg  string // human-readable error (available once status=="error")
	created time.Time
}

var (
	sessionJobs   = map[string]*SessionJob{}
	sessionJobsMu sync.Mutex
)

// SetJobStatus is called by the goroutine in pairing.go to update a session-gen job.
func SetJobStatus(id string, j SessionJob) {
	sessionJobsMu.Lock()
	defer sessionJobsMu.Unlock()
	if existing, ok := sessionJobs[id]; ok {
		existing.Status = j.Status
		existing.Code = j.Code
		existing.Session = j.Session
		existing.ErrMsg = j.ErrMsg
	}
}

func newJobID() string {
	id, err := uuid.NewRandom()
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return id.String()
}

// NewSessionJob registers a new session-gen job and returns its ID.
// Called by pairing.go before starting the background goroutine.
func NewSessionJob() string {
	id := newJobID()
	sessionJobsMu.Lock()
	sessionJobs[id] = &SessionJob{Status: "pending", created: time.Now()}
	sessionJobsMu.Unlock()
	// Auto-expire after 10 minutes in a background goroutine.
	go func() {
		time.Sleep(10 * time.Minute)
		sessionJobsMu.Lock()
		delete(sessionJobs, id)
		sessionJobsMu.Unlock()
	}()
	return id
}

// ErrAlreadyLinked is returned by Hooks.Pair when a session already exists.
var ErrAlreadyLinked = errors.New("a session is already linked — only one session is allowed")

var (
	pairGate sync.Mutex
	lastPair time.Time
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func port() string {
	for _, k := range []string{"PORT", "SERVER_PORT", "WEB_PORT"} {
		if p := os.Getenv(k); p != "" {
			return p
		}
	}
	return "3000"
}

// Start launches the HTTP server in the background.
func Start(h Hooks) {
	needKey := os.Getenv("PAIR_KEY") != ""
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		info := h.Info()
		info.NeedKey = needKey && !info.Paired
		if !info.Paired {
			info.Groups = nil
		}
		writeJSON(w, http.StatusOK, info)
	})

	mux.HandleFunc("/api/pair", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "POST only"})
			return
		}
		if h.Info().Paired {
			writeJSON(w, http.StatusConflict, map[string]interface{}{"ok": false, "paired": true, "error": ErrAlreadyLinked.Error()})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var body struct {
			Number string `json:"number"`
			Key    string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "Invalid request"})
			return
		}
		if k := os.Getenv("PAIR_KEY"); k != "" && strings.TrimSpace(body.Key) != k {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"ok": false, "error": "Wrong access key"})
			return
		}

		// one pairing attempt at a time, at most one every 8 seconds
		pairGate.Lock()
		if wait := 8*time.Second - time.Since(lastPair); wait > 0 {
			pairGate.Unlock()
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{"ok": false, "error": "Please wait a few seconds and try again"})
			return
		}
		lastPair = time.Now()
		pairGate.Unlock()

		code, err := h.Pair(body.Number)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrAlreadyLinked) {
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]interface{}{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "code": code})
	})

	// ── /api/logout  POST → {ok} ─────────────────────────────────────────────
	// Called by the dashboard Logout button. Unlinks the device from WhatsApp,
	// wipes the MongoDB session, and exits the process (Heroku restarts to pair).
	mux.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "POST only"})
			return
		}
		if h.Logout == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"ok": false, "error": "logout not available"})
			return
		}
		if !h.Info().Paired {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "not linked"})
			return
		}
		if err := h.Logout(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "message": "Logged out — bot is restarting"})
	})

	// ── /api/gensession/start  POST {number, key} → {ok, id, code} ──────────
	mux.HandleFunc("/api/gensession/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "POST only"})
			return
		}
		if h.GenSession == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"ok": false, "error": "session gen not available"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var body struct {
			Number string `json:"number"`
			Key    string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "Invalid request"})
			return
		}
		if k := os.Getenv("PAIR_KEY"); k != "" && strings.TrimSpace(body.Key) != k {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"ok": false, "error": "Wrong access key"})
			return
		}

		jobID, code, err := h.GenSession(body.Number)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "id": jobID, "code": code})
	})

	// ── /api/gensession/poll  GET ?id=  → {ok, status, code, session, error} ─
	mux.HandleFunc("/api/gensession/poll", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "missing id"})
			return
		}
		sessionJobsMu.Lock()
		job, found := sessionJobs[id]
		if found && job.Status == "done" {
			// one-time reveal: remove job after session is retrieved
			delete(sessionJobs, id)
		}
		sessionJobsMu.Unlock()

		if !found {
			writeJSON(w, http.StatusNotFound, map[string]interface{}{"ok": false, "error": "job not found or already retrieved"})
			return
		}
		resp := map[string]interface{}{
			"ok":      true,
			"status":  job.Status,
			"code":    job.Code,
			"session": job.Session,
			"error":   job.ErrMsg,
		}
		writeJSON(w, http.StatusOK, resp)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		name := "pair.html"
		if h.Info().Paired {
			name = "dash.html"
		}
		data, err := pages.ReadFile(name)
		if err != nil {
			http.Error(w, "page missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
	})

	srv := &http.Server{
		Addr:              ":" + port(),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("🌐 PBX Elite web page listening on :%s", port())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ web server error: %v", err)
		}
	}()
}
