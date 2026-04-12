package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	yamlpkg "github.com/APICerberus/APICerebrus/internal/pkg/yaml"
)

func (s *Server) exportConfig() (map[string]any, error) {
	s.mu.RLock()
	cfg := config.CloneConfig(s.cfg)
	s.mu.RUnlock()

	yamlBytes, err := yamlpkg.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config yaml: %w", err)
	}
	return map[string]any{
		"config": cfg,
		"yaml":   string(yamlBytes),
	}, nil
}

func (s *Server) swapRuntime(newCfg *config.Config) error {
	if newCfg == nil {
		return errors.New("new config is nil")
	}
	runtimeCfg := config.CloneConfig(newCfg)
	newGateway, newAdmin, err := buildRuntime(runtimeCfg)
	if err != nil {
		return err
	}

	s.mu.Lock()
	oldGateway := s.gateway
	s.cfg = runtimeCfg
	s.gateway = newGateway
	s.admin = newAdmin
	s.adminToken = ""
	s.mu.Unlock()

	if oldGateway != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = oldGateway.Shutdown(ctx)
		cancel()
	}
	return nil
}
