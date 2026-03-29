package admin

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/gateway"
	jsonutil "github.com/APICerberus/APICerebrus/internal/pkg/json"
	"github.com/APICerberus/APICerebrus/internal/pkg/uuid"
	"github.com/APICerberus/APICerebrus/internal/version"
)

// Server hosts Admin REST API endpoints.
type Server struct {
	mu      sync.RWMutex
	cfg     *config.Config
	gateway *gateway.Gateway
	mux     *http.ServeMux

	startedAt time.Time
}

// NewServer initializes admin routes.
func NewServer(cfg *config.Config, gw *gateway.Gateway) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if gw == nil {
		return nil, errors.New("gateway is nil")
	}

	s := &Server{
		cfg:       cfg,
		gateway:   gw,
		mux:       http.NewServeMux(),
		startedAt: time.Now(),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.handle("GET /admin/api/v1/status", s.handleStatus)
	s.handle("GET /admin/api/v1/info", s.handleInfo)
	s.handle("POST /admin/api/v1/config/reload", s.handleConfigReload)

	s.handle("GET /admin/api/v1/services", s.listServices)
	s.handle("POST /admin/api/v1/services", s.createService)
	s.handle("GET /admin/api/v1/services/{id}", s.getService)
	s.handle("PUT /admin/api/v1/services/{id}", s.updateService)
	s.handle("DELETE /admin/api/v1/services/{id}", s.deleteService)

	s.handle("GET /admin/api/v1/routes", s.listRoutes)
	s.handle("POST /admin/api/v1/routes", s.createRoute)
	s.handle("GET /admin/api/v1/routes/{id}", s.getRoute)
	s.handle("PUT /admin/api/v1/routes/{id}", s.updateRoute)
	s.handle("DELETE /admin/api/v1/routes/{id}", s.deleteRoute)

	s.handle("GET /admin/api/v1/upstreams", s.listUpstreams)
	s.handle("POST /admin/api/v1/upstreams", s.createUpstream)
	s.handle("GET /admin/api/v1/upstreams/{id}", s.getUpstream)
	s.handle("PUT /admin/api/v1/upstreams/{id}", s.updateUpstream)
	s.handle("DELETE /admin/api/v1/upstreams/{id}", s.deleteUpstream)
	s.handle("POST /admin/api/v1/upstreams/{id}/targets", s.addUpstreamTarget)
	s.handle("DELETE /admin/api/v1/upstreams/{id}/targets/{tid}", s.deleteUpstreamTarget)
	s.handle("GET /admin/api/v1/upstreams/{id}/health", s.getUpstreamHealth)
}

func (s *Server) handle(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, s.withAdminAuth(handler))
}

func (s *Server) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		expected := s.cfg.Admin.APIKey
		s.mu.RUnlock()

		provided := r.Header.Get("X-Admin-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "admin_unauthorized", "Invalid admin key")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleInfo(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	startedAt := s.startedAt
	s.mu.RUnlock()

	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"version":    version.Version,
		"commit":     version.Commit,
		"build_time": version.BuildTime,
		"uptime_sec": int(time.Since(startedAt).Seconds()),
		"summary": map[string]any{
			"services":  len(cfg.Services),
			"routes":    len(cfg.Routes),
			"upstreams": len(cfg.Upstreams),
		},
	})
}

func (s *Server) handleConfigReload(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	next := cloneConfig(s.cfg)
	s.mu.RUnlock()

	if err := s.gateway.Reload(next); err != nil {
		writeError(w, http.StatusBadRequest, "config_reload_failed", err.Error())
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{"reloaded": true})
}

func (s *Server) listServices(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = jsonutil.WriteJSON(w, http.StatusOK, s.cfg.Services)
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	var in config.Service
	if err := jsonutil.ReadJSON(r, &in, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" {
		id, err := uuid.NewString()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "id_generation_failed", err.Error())
			return
		}
		in.ID = id
	}
	if strings.TrimSpace(in.Protocol) == "" {
		in.Protocol = "http"
	}
	if err := validateServiceInput(in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_service", err.Error())
		return
	}

	if err := s.mutateConfig(func(cfg *config.Config) error {
		if serviceByID(cfg, in.ID) != nil {
			return errors.New("service id already exists")
		}
		if serviceByName(cfg, in.Name) != nil {
			return errors.New("service name already exists")
		}
		if !upstreamExists(cfg, in.Upstream) {
			return errors.New("referenced upstream does not exist")
		}
		cfg.Services = append(cfg.Services, in)
		return nil
	}); err != nil {
		writeError(w, http.StatusBadRequest, "create_service_failed", err.Error())
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, in)
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	defer s.mu.RUnlock()
	svc := serviceByID(s.cfg, id)
	if svc == nil {
		writeError(w, http.StatusNotFound, "service_not_found", "Service not found")
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, svc)
}

func (s *Server) updateService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in config.Service
	if err := jsonutil.ReadJSON(r, &in, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" {
		in.ID = id
	}
	if in.ID != id {
		writeError(w, http.StatusBadRequest, "invalid_service", "path id and payload id must match")
		return
	}
	if strings.TrimSpace(in.Protocol) == "" {
		in.Protocol = "http"
	}
	if err := validateServiceInput(in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_service", err.Error())
		return
	}

	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := serviceIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("service not found")
		}
		if !upstreamExists(cfg, in.Upstream) {
			return errors.New("referenced upstream does not exist")
		}
		for i := range cfg.Services {
			if i != idx && strings.EqualFold(cfg.Services[i].Name, in.Name) {
				return errors.New("service name already exists")
			}
		}
		cfg.Services[idx] = in
		return nil
	}); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "service not found" {
			status = http.StatusNotFound
		}
		writeError(w, status, "update_service_failed", err.Error())
		return
	}

	_ = jsonutil.WriteJSON(w, http.StatusOK, in)
}

func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := serviceIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("service not found")
		}
		svc := cfg.Services[idx]
		for _, rt := range cfg.Routes {
			if rt.Service == svc.ID || rt.Service == svc.Name {
				return errors.New("service is referenced by route")
			}
		}
		cfg.Services = append(cfg.Services[:idx], cfg.Services[idx+1:]...)
		return nil
	}); err != nil {
		switch err.Error() {
		case "service not found":
			writeError(w, http.StatusNotFound, "service_not_found", err.Error())
		case "service is referenced by route":
			writeError(w, http.StatusConflict, "service_in_use", err.Error())
		default:
			writeError(w, http.StatusBadRequest, "delete_service_failed", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listRoutes(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = jsonutil.WriteJSON(w, http.StatusOK, s.cfg.Routes)
}

func (s *Server) createRoute(w http.ResponseWriter, r *http.Request) {
	var in config.Route
	if err := jsonutil.ReadJSON(r, &in, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" {
		id, err := uuid.NewString()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "id_generation_failed", err.Error())
			return
		}
		in.ID = id
	}
	if len(in.Methods) == 0 {
		in.Methods = []string{http.MethodGet}
	}
	if err := validateRouteInput(in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_route", err.Error())
		return
	}

	if err := s.mutateConfig(func(cfg *config.Config) error {
		if routeByID(cfg, in.ID) != nil {
			return errors.New("route id already exists")
		}
		if routeByName(cfg, in.Name) != nil {
			return errors.New("route name already exists")
		}
		if !serviceExists(cfg, in.Service) {
			return errors.New("referenced service does not exist")
		}
		cfg.Routes = append(cfg.Routes, in)
		return nil
	}); err != nil {
		writeError(w, http.StatusBadRequest, "create_route_failed", err.Error())
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, in)
}

func (s *Server) getRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	defer s.mu.RUnlock()
	route := routeByID(s.cfg, id)
	if route == nil {
		writeError(w, http.StatusNotFound, "route_not_found", "Route not found")
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, route)
}

func (s *Server) updateRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in config.Route
	if err := jsonutil.ReadJSON(r, &in, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" {
		in.ID = id
	}
	if in.ID != id {
		writeError(w, http.StatusBadRequest, "invalid_route", "path id and payload id must match")
		return
	}
	if len(in.Methods) == 0 {
		in.Methods = []string{http.MethodGet}
	}
	if err := validateRouteInput(in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_route", err.Error())
		return
	}

	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := routeIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("route not found")
		}
		if !serviceExists(cfg, in.Service) {
			return errors.New("referenced service does not exist")
		}
		for i := range cfg.Routes {
			if i != idx && strings.EqualFold(cfg.Routes[i].Name, in.Name) {
				return errors.New("route name already exists")
			}
		}
		cfg.Routes[idx] = in
		return nil
	}); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "route not found" {
			status = http.StatusNotFound
		}
		writeError(w, status, "update_route_failed", err.Error())
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, in)
}

func (s *Server) deleteRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := routeIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("route not found")
		}
		cfg.Routes = append(cfg.Routes[:idx], cfg.Routes[idx+1:]...)
		return nil
	}); err != nil {
		if err.Error() == "route not found" {
			writeError(w, http.StatusNotFound, "route_not_found", err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "delete_route_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listUpstreams(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_ = jsonutil.WriteJSON(w, http.StatusOK, s.cfg.Upstreams)
}

func (s *Server) createUpstream(w http.ResponseWriter, r *http.Request) {
	var in config.Upstream
	if err := jsonutil.ReadJSON(r, &in, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" {
		id, err := uuid.NewString()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "id_generation_failed", err.Error())
			return
		}
		in.ID = id
	}
	if strings.TrimSpace(in.Algorithm) == "" {
		in.Algorithm = "round_robin"
	}
	if err := validateUpstreamInput(in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_upstream", err.Error())
		return
	}

	if err := s.mutateConfig(func(cfg *config.Config) error {
		if upstreamByID(cfg, in.ID) != nil {
			return errors.New("upstream id already exists")
		}
		if upstreamByName(cfg, in.Name) != nil {
			return errors.New("upstream name already exists")
		}
		cfg.Upstreams = append(cfg.Upstreams, in)
		return nil
	}); err != nil {
		writeError(w, http.StatusBadRequest, "create_upstream_failed", err.Error())
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, in)
}

func (s *Server) getUpstream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	defer s.mu.RUnlock()
	up := upstreamByID(s.cfg, id)
	if up == nil {
		writeError(w, http.StatusNotFound, "upstream_not_found", "Upstream not found")
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, up)
}

func (s *Server) updateUpstream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in config.Upstream
	if err := jsonutil.ReadJSON(r, &in, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" {
		in.ID = id
	}
	if in.ID != id {
		writeError(w, http.StatusBadRequest, "invalid_upstream", "path id and payload id must match")
		return
	}
	if strings.TrimSpace(in.Algorithm) == "" {
		in.Algorithm = "round_robin"
	}
	if err := validateUpstreamInput(in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_upstream", err.Error())
		return
	}

	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := upstreamIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("upstream not found")
		}
		for i := range cfg.Upstreams {
			if i != idx && strings.EqualFold(cfg.Upstreams[i].Name, in.Name) {
				return errors.New("upstream name already exists")
			}
		}
		cfg.Upstreams[idx] = in
		return nil
	}); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "upstream not found" {
			status = http.StatusNotFound
		}
		writeError(w, status, "update_upstream_failed", err.Error())
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, in)
}

func (s *Server) deleteUpstream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := upstreamIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("upstream not found")
		}
		up := cfg.Upstreams[idx]
		for _, svc := range cfg.Services {
			if svc.Upstream == up.ID || svc.Upstream == up.Name {
				return errors.New("upstream is referenced by service")
			}
		}
		cfg.Upstreams = append(cfg.Upstreams[:idx], cfg.Upstreams[idx+1:]...)
		return nil
	}); err != nil {
		switch err.Error() {
		case "upstream not found":
			writeError(w, http.StatusNotFound, "upstream_not_found", err.Error())
		case "upstream is referenced by service":
			writeError(w, http.StatusConflict, "upstream_in_use", err.Error())
		default:
			writeError(w, http.StatusBadRequest, "delete_upstream_failed", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) addUpstreamTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in config.UpstreamTarget
	if err := jsonutil.ReadJSON(r, &in, 1<<20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_payload", err.Error())
		return
	}
	if strings.TrimSpace(in.ID) == "" {
		generated, err := uuid.NewString()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "id_generation_failed", err.Error())
			return
		}
		in.ID = generated
	}
	if strings.TrimSpace(in.Address) == "" {
		writeError(w, http.StatusBadRequest, "invalid_target", "target address is required")
		return
	}
	if in.Weight <= 0 {
		in.Weight = 100
	}

	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := upstreamIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("upstream not found")
		}
		for _, t := range cfg.Upstreams[idx].Targets {
			if t.ID == in.ID {
				return errors.New("target id already exists")
			}
		}
		cfg.Upstreams[idx].Targets = append(cfg.Upstreams[idx].Targets, in)
		return nil
	}); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "upstream not found" {
			status = http.StatusNotFound
		}
		writeError(w, status, "add_target_failed", err.Error())
		return
	}
	_ = jsonutil.WriteJSON(w, http.StatusCreated, in)
}

func (s *Server) deleteUpstreamTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	targetID := r.PathValue("tid")

	if err := s.mutateConfig(func(cfg *config.Config) error {
		idx := upstreamIndexByID(cfg, id)
		if idx < 0 {
			return errors.New("upstream not found")
		}
		targets := cfg.Upstreams[idx].Targets
		for i := range targets {
			if targets[i].ID == targetID {
				cfg.Upstreams[idx].Targets = append(targets[:i], targets[i+1:]...)
				return nil
			}
		}
		return errors.New("target not found")
	}); err != nil {
		switch err.Error() {
		case "upstream not found":
			writeError(w, http.StatusNotFound, "upstream_not_found", err.Error())
		case "target not found":
			writeError(w, http.StatusNotFound, "target_not_found", err.Error())
		default:
			writeError(w, http.StatusBadRequest, "delete_target_failed", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getUpstreamHealth(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	up := upstreamByID(s.cfg, id)
	s.mu.RUnlock()
	if up == nil {
		writeError(w, http.StatusNotFound, "upstream_not_found", "Upstream not found")
		return
	}

	health := s.gateway.UpstreamHealth(up.Name)
	targets := make([]map[string]any, 0, len(up.Targets))
	for _, t := range up.Targets {
		targets = append(targets, map[string]any{
			"id":      t.ID,
			"address": t.Address,
			"healthy": health[t.ID],
		})
	}
	_ = jsonutil.WriteJSON(w, http.StatusOK, map[string]any{
		"upstream_id":   up.ID,
		"upstream_name": up.Name,
		"targets":       targets,
	})
}

func (s *Server) mutateConfig(mutator func(*config.Config) error) error {
	s.mu.RLock()
	next := cloneConfig(s.cfg)
	s.mu.RUnlock()

	if err := mutator(next); err != nil {
		return err
	}
	if err := s.gateway.Reload(next); err != nil {
		return err
	}

	s.mu.Lock()
	s.cfg = next
	s.mu.Unlock()
	return nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	_ = jsonutil.WriteJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func validateServiceInput(svc config.Service) error {
	if strings.TrimSpace(svc.Name) == "" {
		return errors.New("service name is required")
	}
	if strings.TrimSpace(svc.Upstream) == "" {
		return errors.New("service upstream is required")
	}
	switch strings.ToLower(strings.TrimSpace(svc.Protocol)) {
	case "http", "grpc", "graphql":
	default:
		return errors.New("service protocol must be http, grpc, or graphql")
	}
	return nil
}

func validateRouteInput(route config.Route) error {
	if strings.TrimSpace(route.Name) == "" {
		return errors.New("route name is required")
	}
	if strings.TrimSpace(route.Service) == "" {
		return errors.New("route service is required")
	}
	if len(route.Paths) == 0 {
		return errors.New("route must define at least one path")
	}
	return nil
}

func validateUpstreamInput(up config.Upstream) error {
	if strings.TrimSpace(up.Name) == "" {
		return errors.New("upstream name is required")
	}
	if len(up.Targets) == 0 {
		return errors.New("upstream must include at least one target")
	}
	for _, t := range up.Targets {
		if strings.TrimSpace(t.ID) == "" {
			return errors.New("upstream target id is required")
		}
		if strings.TrimSpace(t.Address) == "" {
			return errors.New("upstream target address is required")
		}
		if t.Weight <= 0 {
			return errors.New("upstream target weight must be greater than zero")
		}
	}
	return nil
}

func cloneConfig(src *config.Config) *config.Config {
	if src == nil {
		return &config.Config{}
	}
	out := *src
	out.Services = append([]config.Service(nil), src.Services...)
	out.Routes = append([]config.Route(nil), src.Routes...)
	out.GlobalPlugins = clonePluginConfigs(src.GlobalPlugins)
	for i := range out.Routes {
		out.Routes[i].Plugins = clonePluginConfigs(src.Routes[i].Plugins)
	}

	out.Upstreams = append([]config.Upstream(nil), src.Upstreams...)
	for i := range out.Upstreams {
		out.Upstreams[i].Targets = append([]config.UpstreamTarget(nil), src.Upstreams[i].Targets...)
	}

	out.Consumers = append([]config.Consumer(nil), src.Consumers...)
	for i := range out.Consumers {
		out.Consumers[i].APIKeys = append([]config.ConsumerAPIKey(nil), src.Consumers[i].APIKeys...)
		out.Consumers[i].ACLGroups = append([]string(nil), src.Consumers[i].ACLGroups...)
		if src.Consumers[i].Metadata != nil {
			out.Consumers[i].Metadata = make(map[string]any, len(src.Consumers[i].Metadata))
			for k, v := range src.Consumers[i].Metadata {
				out.Consumers[i].Metadata[k] = v
			}
		}
	}
	out.Auth.APIKey.KeyNames = append([]string(nil), src.Auth.APIKey.KeyNames...)
	out.Auth.APIKey.QueryNames = append([]string(nil), src.Auth.APIKey.QueryNames...)
	out.Auth.APIKey.CookieNames = append([]string(nil), src.Auth.APIKey.CookieNames...)
	return &out
}

func clonePluginConfigs(in []config.PluginConfig) []config.PluginConfig {
	if len(in) == 0 {
		return nil
	}
	out := make([]config.PluginConfig, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].Enabled != nil {
			v := *in[i].Enabled
			out[i].Enabled = &v
		}
		if in[i].Config != nil {
			out[i].Config = make(map[string]any, len(in[i].Config))
			for k, v := range in[i].Config {
				out[i].Config[k] = v
			}
		}
	}
	return out
}

func serviceByID(cfg *config.Config, id string) *config.Service {
	for i := range cfg.Services {
		if cfg.Services[i].ID == id {
			return &cfg.Services[i]
		}
	}
	return nil
}

func serviceByName(cfg *config.Config, name string) *config.Service {
	for i := range cfg.Services {
		if strings.EqualFold(cfg.Services[i].Name, name) {
			return &cfg.Services[i]
		}
	}
	return nil
}

func serviceIndexByID(cfg *config.Config, id string) int {
	for i := range cfg.Services {
		if cfg.Services[i].ID == id {
			return i
		}
	}
	return -1
}

func routeByID(cfg *config.Config, id string) *config.Route {
	for i := range cfg.Routes {
		if cfg.Routes[i].ID == id {
			return &cfg.Routes[i]
		}
	}
	return nil
}

func routeByName(cfg *config.Config, name string) *config.Route {
	for i := range cfg.Routes {
		if strings.EqualFold(cfg.Routes[i].Name, name) {
			return &cfg.Routes[i]
		}
	}
	return nil
}

func routeIndexByID(cfg *config.Config, id string) int {
	for i := range cfg.Routes {
		if cfg.Routes[i].ID == id {
			return i
		}
	}
	return -1
}

func upstreamByID(cfg *config.Config, id string) *config.Upstream {
	for i := range cfg.Upstreams {
		if cfg.Upstreams[i].ID == id {
			return &cfg.Upstreams[i]
		}
	}
	return nil
}

func upstreamByName(cfg *config.Config, name string) *config.Upstream {
	for i := range cfg.Upstreams {
		if strings.EqualFold(cfg.Upstreams[i].Name, name) {
			return &cfg.Upstreams[i]
		}
	}
	return nil
}

func upstreamIndexByID(cfg *config.Config, id string) int {
	for i := range cfg.Upstreams {
		if cfg.Upstreams[i].ID == id {
			return i
		}
	}
	return -1
}

func upstreamExists(cfg *config.Config, nameOrID string) bool {
	return upstreamByID(cfg, nameOrID) != nil || upstreamByName(cfg, nameOrID) != nil
}

func serviceExists(cfg *config.Config, nameOrID string) bool {
	return serviceByID(cfg, nameOrID) != nil || serviceByName(cfg, nameOrID) != nil
}
