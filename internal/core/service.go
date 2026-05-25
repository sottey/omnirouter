package core

import (
	"fmt"
	"time"

	omnirouterconfig "omnirouter/internal/config"
)

type Service struct {
	configPath string
	config     *omnirouterconfig.Config
}

func NewService(configPath string, cfg *omnirouterconfig.Config) *Service {
	return &Service{
		configPath: configPath,
		config:     cfg,
	}
}

func (s *Service) GetConfig() (omnirouterconfig.Config, error) {
	if s.config == nil {
		return omnirouterconfig.Config{}, fmt.Errorf("config not loaded")
	}

	return *s.config, nil
}

func (s *Service) SaveConfig(cfg omnirouterconfig.Config) error {
	if err := cfg.ApplyDefaultsAndValidate(); err != nil {
		return err
	}

	if err := omnirouterconfig.WriteConfig(s.configPath, &cfg); err != nil {
		return err
	}

	s.config = &cfg
	return nil
}

func (s *Service) SendPrompt(targetName string, prompt string) (omnirouterconfig.SendPromptResult, error) {
	if s.config == nil {
		return omnirouterconfig.SendPromptResult{}, fmt.Errorf("config not loaded")
	}

	target, chosenTargetName, err := ResolveTarget(s.config, targetName, prompt)
	if err != nil {
		return omnirouterconfig.SendPromptResult{}, err
	}

	if target.Type == "api" {
		responseText, err := CallLLMAPI(target.Provider, target.Model, target.APIKeyEnv, target.SystemPrompt, prompt)
		if err != nil {
			return omnirouterconfig.SendPromptResult{}, err
		}
		return omnirouterconfig.SendPromptResult{
			TargetName:   chosenTargetName,
			ResponseText: responseText,
			IsAPI:        true,
		}, nil
	}

	if err := SendPromptToTarget(*target, prompt); err != nil {
		return omnirouterconfig.SendPromptResult{}, err
	}

	return omnirouterconfig.SendPromptResult{
		TargetName: chosenTargetName,
		IsAPI:      false,
	}, nil
}

func (s *Service) TestTarget(targetName string) error {
	if s.config == nil {
		return fmt.Errorf("config not loaded")
	}

	target, err := s.config.FindTargetByName(targetName)
	if err != nil {
		return err
	}
	if target.Type == "auto" {
		return fmt.Errorf("test target must not be auto")
	}

	testPrompt := fmt.Sprintf("OmniRouter test message (%s)", time.Now().Format(time.RFC3339))
	return SendPromptToTarget(*target, testPrompt)
}

func (s *Service) ReloadConfig() error {
	cfg, err := omnirouterconfig.LoadConfig(s.configPath)
	if err != nil {
		return err
	}

	s.config = cfg
	return nil
}
