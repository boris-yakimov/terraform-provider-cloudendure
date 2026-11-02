// Package fakeapi implements an in-memory fake of the CloudEndure API for tests.
//
// It mimics the parts of the real API the provider relies on, including the
// session cookie / XSRF token handshake with a backslash-escaped cookie value.
package fakeapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
)

const (
	// XSRFToken is the token issued by the fake server (without quotes).
	XSRFToken = `fakeXsrf==\012`
	session   = "fake-session"
)

// Server is a fake CloudEndure API server.
type Server struct {
	*httptest.Server

	APIToken  string
	ProjectID string

	mu            sync.Mutex
	nextID        int
	Machines      map[string]*client.Machine
	Blueprints    map[string]*client.Blueprint
	RecoveryPlans map[string]*client.RecoveryPlan
	Logins        int
}

// New starts a fake server accepting apiToken for project projectID.
// The caller must Close it.
func New(apiToken, projectID string) *Server {
	s := &Server{
		APIToken:      apiToken,
		ProjectID:     projectID,
		Machines:      map[string]*client.Machine{},
		Blueprints:    map[string]*client.Blueprint{},
		RecoveryPlans: map[string]*client.RecoveryPlan{},
	}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

// BaseURL returns the API base URL to configure clients with.
func (s *Server) BaseURL() string { return s.URL + "/api/latest" }

// AddMachine registers a source machine and returns its ID.
func (s *Server) AddMachine(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.newIDLocked("machine")
	s.Machines[id] = &client.Machine{
		ID:                id,
		ReplicationStatus: "STARTED",
		SourceProperties:  client.MachineSourceProperties{Name: name, OS: "linux"},
	}
	return id
}

// AddBlueprint stores bp (assigning an ID if empty) and returns its ID.
func (s *Server) AddBlueprint(bp client.Blueprint) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if bp.ID == "" {
		bp.ID = s.newIDLocked("blueprint")
	}
	s.Blueprints[bp.ID] = &bp
	return bp.ID
}

func (s *Server) newIDLocked(prefix string) string {
	s.nextID++
	return fmt.Sprintf("%s-%04d", prefix, s.nextID)
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	path, ok := strings.CutPrefix(r.URL.Path, "/api/latest")
	if !ok {
		http.NotFound(w, r)
		return
	}

	if path == "/login" && r.Method == http.MethodPost {
		s.login(w, r)
		return
	}

	if !s.authenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[0] != "projects" {
		http.NotFound(w, r)
		return
	}
	if parts[1] != s.ProjectID {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := ""
	if len(parts) == 4 {
		id = parts[3]
	} else if len(parts) > 4 {
		http.NotFound(w, r)
		return
	}

	switch parts[2] {
	case "machines":
		s.machines(w, r, id)
	case "blueprints":
		s.blueprints(w, r, id)
	case "recoveryPlans":
		s.recoveryPlans(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserAPIToken string `json:"userApiToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserAPIToken != s.APIToken {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	s.mu.Lock()
	s.Logins++
	s.mu.Unlock()

	// Set the headers by hand: http.SetCookie would refuse the backslash.
	w.Header().Add("Set-Cookie", `XSRF-TOKEN="`+XSRFToken+`"; Path=/`)
	w.Header().Add("Set-Cookie", "session="+session+"; HttpOnly; Path=/")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `{"username":"fake"}`)
}

func (s *Server) authenticated(r *http.Request) bool {
	if r.Header.Get("X-XSRF-TOKEN") != XSRFToken {
		return false
	}
	cookie := r.Header.Get("Cookie")
	return strings.Contains(cookie, "session="+session) &&
		strings.Contains(cookie, `XSRF-TOKEN="`+XSRFToken+`"`)
}

func (s *Server) machines(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if id == "" {
		writeList(w, s.Machines)
		return
	}
	m, ok := s.Machines[id]
	if !ok {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) blueprints(w http.ResponseWriter, r *http.Request, id string) {
	switch {
	case id == "" && r.Method == http.MethodGet:
		writeList(w, s.Blueprints)

	case id == "" && r.Method == http.MethodPost:
		var bp client.Blueprint
		if !decode(w, r, &bp) {
			return
		}
		if _, ok := s.Machines[bp.MachineID]; !ok {
			writeError(w, http.StatusBadRequest, "unknown machineId")
			return
		}
		for _, existing := range s.Blueprints {
			if existing.MachineID == bp.MachineID {
				writeError(w, http.StatusConflict, "machine already has a blueprint")
				return
			}
		}
		bp.ID = s.newIDLocked("blueprint")
		s.applyBlueprintDefaults(&bp)
		s.Blueprints[bp.ID] = &bp
		writeJSON(w, http.StatusCreated, bp)

	case id != "" && r.Method == http.MethodGet:
		bp, ok := s.Blueprints[id]
		if !ok {
			writeError(w, http.StatusNotFound, "blueprint not found")
			return
		}
		writeJSON(w, http.StatusOK, bp)

	case id != "" && r.Method == http.MethodPatch:
		bp, ok := s.Blueprints[id]
		if !ok {
			writeError(w, http.StatusNotFound, "blueprint not found")
			return
		}
		// Merge the patch onto the stored object, like the real API.
		if !decode(w, r, bp) {
			return
		}
		bp.ID = id
		writeJSON(w, http.StatusOK, bp)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// applyBlueprintDefaults fills computed fields the real API returns.
func (s *Server) applyBlueprintDefaults(bp *client.Blueprint) {
	if bp.Region == "" {
		bp.Region = "region-0001"
	}
	if bp.RecommendedInstanceType == "" {
		bp.RecommendedInstanceType = "t3.medium"
	}
	if m, ok := s.Machines[bp.MachineID]; ok && bp.MachineName == "" {
		bp.MachineName = m.SourceProperties.Name
	}
}

func (s *Server) recoveryPlans(w http.ResponseWriter, r *http.Request, id string) {
	switch {
	case id == "" && r.Method == http.MethodGet:
		writeList(w, s.RecoveryPlans)

	case id == "" && r.Method == http.MethodPost:
		var rp client.RecoveryPlan
		if !decode(w, r, &rp) {
			return
		}
		rp.ID = s.newIDLocked("plan")
		s.RecoveryPlans[rp.ID] = &rp
		writeJSON(w, http.StatusCreated, rp)

	case id != "" && r.Method == http.MethodGet:
		rp, ok := s.RecoveryPlans[id]
		if !ok {
			writeError(w, http.StatusNotFound, "recovery plan not found")
			return
		}
		writeJSON(w, http.StatusOK, rp)

	case id != "" && r.Method == http.MethodPatch:
		rp, ok := s.RecoveryPlans[id]
		if !ok {
			writeError(w, http.StatusNotFound, "recovery plan not found")
			return
		}
		var patch client.RecoveryPlan
		if !decode(w, r, &patch) {
			return
		}
		patch.ID = rp.ID
		s.RecoveryPlans[id] = &patch
		writeJSON(w, http.StatusOK, patch)

	case id != "" && r.Method == http.MethodDelete:
		if _, ok := s.RecoveryPlans[id]; !ok {
			writeError(w, http.StatusNotFound, "recovery plan not found")
			return
		}
		delete(s.RecoveryPlans, id)
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return false
	}
	return true
}

func writeList[T any](w http.ResponseWriter, items map[string]*T) {
	list := make([]*T, 0, len(items))
	for _, v := range items {
		list = append(list, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"message": msg})
}
