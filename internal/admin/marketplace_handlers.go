package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	jsonutil "github.com/APICerberus/APICerebrus/internal/pkg/json"
	"github.com/APICerberus/APICerebrus/internal/plugin"
)

func (s *Server) handleMarketplaceSearch(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	query := r.URL.Query().Get("q")
	tags := r.URL.Query()["tag"]

	results := mp.Search(query, tags)
	if results == nil {
		results = []plugin.PluginListing{}
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"results": results,
		"count":   len(results),
	})
}

func (s *Server) handleMarketplaceGetPlugin(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	id := r.PathValue("id")
	listing, err := mp.GetPlugin(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "plugin_not_found", err.Error())
		return
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, listing)
}

func (s *Server) handleMarketplaceInstall(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	var req struct {
		ID      string `json:"id"`
		Version string `json:"version,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "plugin id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	installed, err := mp.Install(ctx, req.ID, req.Version)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "context deadline exceeded" {
			status = http.StatusGatewayTimeout
		}
		writeError(w, status, "install_failed", err.Error())
		return
	}

	_ = jsonutil.WriteJSON(w, http.StatusCreated, installed)
}

func (s *Server) handleMarketplaceUninstall(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	id := r.PathValue("id")
	if err := mp.Uninstall(id); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "plugin not installed: "+id {
			status = http.StatusNotFound
		}
		writeError(w, status, "uninstall_failed", err.Error())
		return
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "uninstalled", "id": id})
}

func (s *Server) handleMarketplaceListInstalled(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	installed, err := mp.ListInstalled()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	if installed == nil {
		installed = []plugin.InstalledPlugin{}
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"plugins": installed,
		"count":   len(installed),
	})
}

func (s *Server) handleMarketplaceEnable(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	id := r.PathValue("id")
	if err := mp.Enable(id); err != nil {
		writeError(w, http.StatusBadRequest, "enable_failed", err.Error())
		return
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "enabled", "id": id})
}

func (s *Server) handleMarketplaceDisableHandler(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	id := r.PathValue("id")
	if err := mp.Disable(id); err != nil {
		writeError(w, http.StatusBadRequest, "disable_failed", err.Error())
		return
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "disabled", "id": id})
}

func (s *Server) handleMarketplaceCheckUpdates(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	updates, err := mp.CheckForUpdates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "check_updates_failed", err.Error())
		return
	}
	if updates == nil {
		updates = []plugin.PluginUpdate{}
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"updates": updates,
		"count":   len(updates),
	})
}

func (s *Server) handleMarketplaceUpdateIndex(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	if mp == nil || !mp.IsEnabled() {
		writeError(w, http.StatusServiceUnavailable, "marketplace_disabled", "Plugin marketplace is not enabled")
		return
	}

	if err := mp.UpdateIndex(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "update_index_failed", err.Error())
		return
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]string{"status": "index_updated"})
}

func (s *Server) handleMarketplaceStatus(w http.ResponseWriter, r *http.Request) {
	mp := s.ensureMarketplace()
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"enabled": mp != nil && mp.IsEnabled(),
	})
}

// ensureMarketplace lazily initializes the marketplace from config.
func (s *Server) ensureMarketplace() *plugin.Marketplace {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.marketplace != nil {
		return s.marketplace
	}

	cfg := s.cfg.PluginMarketplace
	if !cfg.Enabled {
		return nil
	}

	mpCfg := plugin.MarketplaceConfig{
		Enabled:           cfg.Enabled,
		DataDir:           cfg.DataDir,
		RegistryURL:       cfg.RegistryURL,
		TrustedSigners:    cfg.TrustedSigners,
		TrustedSignerKeys: cfg.TrustedSignerKeys,
		AutoUpdate:        cfg.AutoUpdate,
		UpdateInterval:    cfg.UpdateInterval,
		VerifySignatures:  cfg.VerifySignatures,
		MaxPluginSize:     cfg.MaxPluginSize,
		AllowedPhases:     cfg.AllowedPhases,
	}

	mp, err := plugin.NewMarketplace(mpCfg)
	if err == nil {
		s.marketplace = mp
		return mp
	}
	return nil
}
