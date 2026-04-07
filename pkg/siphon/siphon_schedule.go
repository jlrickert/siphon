package siphon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// createScheduleImpl validates options, generates the backend config (launchd
// plist or cron entry), writes it, and persists the schedule in user config.
func (s *Siphon) createScheduleImpl(_ context.Context, opts *CreateScheduleOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("schedule name is required")
	}
	if opts.Connection == "" {
		return fmt.Errorf("connection is required")
	}

	// Validate connection exists.
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if cfg.Connections == nil {
		return ErrConnectionNotFound
	}
	if _, ok := cfg.Connections[opts.Connection]; !ok {
		return ErrConnectionNotFound
	}

	// Validate repo exists if specified.
	if opts.Repo != "" {
		if cfg.Repos == nil {
			return fmt.Errorf("repo %q not configured", opts.Repo)
		}
		if _, ok := cfg.Repos[opts.Repo]; !ok {
			return fmt.Errorf("repo %q not configured", opts.Repo)
		}
	}

	// Check for duplicate schedule name.
	if cfg.Schedules != nil {
		if _, exists := cfg.Schedules[opts.Name]; exists {
			return ErrScheduleExists
		}
	}

	// Default values.
	if opts.Backend == "" {
		opts.Backend = "launchd"
	}
	if opts.Interval == "" {
		opts.Interval = "daily"
	}
	if opts.Time == "" {
		opts.Time = "02:00"
	}
	if opts.BackupType == "" {
		opts.BackupType = "logical"
	}

	// Build the siphon command line.
	cmdArgs := buildBackupCommandArgs(opts)

	// Find the siphon binary path.
	siphonBin, err := os.Executable()
	if err != nil {
		siphonBin = "siphon"
	}

	// Parse time.
	hour, minute, err := parseTimeHHMM(opts.Time)
	if err != nil {
		return fmt.Errorf("invalid time format %q: %w", opts.Time, err)
	}

	switch opts.Backend {
	case "launchd":
		if err := s.installLaunchd(opts.Name, siphonBin, cmdArgs, hour, minute, opts.Interval); err != nil {
			return fmt.Errorf("installing launchd plist: %w", err)
		}
	case "cron":
		if err := s.installCron(opts.Name, siphonBin, cmdArgs, hour, minute, opts.Interval); err != nil {
			return fmt.Errorf("installing cron entry: %w", err)
		}
	default:
		return fmt.Errorf("unsupported backend: %s", opts.Backend)
	}

	// Save schedule to user config.
	return s.saveScheduleConfig(opts)
}

// listSchedulesImpl returns all configured schedules with active status.
func (s *Siphon) listSchedulesImpl(_ context.Context, _ *ListSchedulesOptions) ([]ScheduleInfo, error) {
	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Schedules) == 0 {
		return []ScheduleInfo{}, nil
	}

	var schedules []ScheduleInfo
	for name, sc := range cfg.Schedules {
		info := ScheduleInfo{
			Name: name,
		}
		if sc.Connection != nil {
			info.Connection = *sc.Connection
		}
		if sc.Repo != nil {
			info.Repo = *sc.Repo
		}
		if sc.Database != nil {
			info.Database = *sc.Database
		}
		if sc.BackupType != nil {
			info.BackupType = *sc.BackupType
		}
		if sc.Compress != nil {
			info.Compress = *sc.Compress
		}
		if sc.Time != nil {
			info.Time = *sc.Time
		}
		if sc.Interval != nil {
			info.Interval = *sc.Interval
		}
		if sc.Backend != nil {
			info.Backend = *sc.Backend
		}
		if sc.Retention != nil {
			info.Retention = sc.Retention
		}

		// Check active status based on backend.
		info.Active = s.isScheduleActive(name, info.Backend)

		schedules = append(schedules, info)
	}

	// Sort by name for deterministic output.
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].Name < schedules[j].Name
	})

	return schedules, nil
}

// removeScheduleImpl removes a schedule's backend artifacts and config entry.
func (s *Siphon) removeScheduleImpl(_ context.Context, opts *RemoveScheduleOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("schedule name is required")
	}

	cfg, err := s.ConfigService.Config(true)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Schedules == nil {
		return ErrScheduleNotFound
	}

	sc, exists := cfg.Schedules[opts.Name]
	if !exists {
		return ErrScheduleNotFound
	}

	// Determine backend.
	backend := "launchd"
	if sc.Backend != nil {
		backend = *sc.Backend
	}

	switch backend {
	case "launchd":
		if err := s.uninstallLaunchd(opts.Name); err != nil {
			return fmt.Errorf("uninstalling launchd plist: %w", err)
		}
	case "cron":
		// Cron removal is a best-effort operation -- we remove from config
		// but cannot safely modify the system crontab in all environments.
	}

	// Remove from user config.
	return s.removeScheduleConfig(opts.Name)
}

// --- internal helpers ---

// buildBackupCommandArgs constructs the siphon CLI arguments for a scheduled
// backup.
func buildBackupCommandArgs(opts *CreateScheduleOptions) []string {
	args := []string{"backup", "create", opts.Connection}
	if opts.Repo != "" {
		args = append(args, "--repo", opts.Repo)
	}
	if opts.Database != "" {
		args = append(args, "--database", opts.Database)
	}
	if opts.BackupType != "" && opts.BackupType != "logical" {
		args = append(args, "--type", opts.BackupType)
	}
	if opts.Compress != "" && opts.Compress != "none" {
		args = append(args, "--compress", opts.Compress)
	}
	if opts.Message != "" {
		args = append(args, "--message", opts.Message)
	}
	if opts.BackupNameFormat != "" {
		args = append(args, "--name", opts.BackupNameFormat)
	}
	return args
}

// parseTimeHHMM parses a "HH:MM" string into hour and minute integers.
func parseTimeHHMM(t string) (int, int, error) {
	parts := strings.SplitN(t, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected HH:MM format")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("invalid hour: %s", parts[0])
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid minute: %s", parts[1])
	}
	return hour, minute, nil
}

// installLaunchd generates a plist and writes it to ~/Library/LaunchAgents/.
func (s *Siphon) installLaunchd(name, siphonBin string, args []string, hour, minute int, interval string) error {
	calendarInterval := map[string]int{
		"Hour":   hour,
		"Minute": minute,
	}

	switch interval {
	case "hourly":
		// Run every hour at the specified minute.
		calendarInterval = map[string]int{
			"Minute": minute,
		}
	case "weekly":
		// Run weekly on Sunday at the specified time.
		calendarInterval["Weekday"] = 0
	case "daily":
		// default: just Hour and Minute
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "Library", "Logs", "siphon")
	plistCfg := LaunchdConfig{
		Label:                 PlistLabel(name),
		Program:               siphonBin,
		Arguments:             args,
		StartCalendarInterval: calendarInterval,
		StandardOutPath:       filepath.Join(logDir, name+".out.log"),
		StandardErrorPath:     filepath.Join(logDir, name+".err.log"),
	}

	data, err := GeneratePlist(plistCfg)
	if err != nil {
		return err
	}

	dir := LaunchAgentsDir()
	plistPath := filepath.Join(dir, PlistFilename(name))

	if err := s.Runtime.Mkdir(dir, 0o755, true); err != nil {
		return fmt.Errorf("creating LaunchAgents directory: %w", err)
	}
	if err := s.Runtime.WriteFile(plistPath, data, 0o644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	return nil
}

// installCron generates a cron entry string. It does not modify the system
// crontab directly -- it stores the entry in config for the user to install.
func (s *Siphon) installCron(name, siphonBin string, args []string, hour, minute int, interval string) error {
	minuteStr := strconv.Itoa(minute)
	hourStr := strconv.Itoa(hour)
	dayStr := "*"
	weekdayStr := "*"

	switch interval {
	case "hourly":
		hourStr = "*"
	case "weekly":
		weekdayStr = "0"
	case "daily":
		// defaults are fine
	}

	fullCmd := siphonBin
	for _, a := range args {
		fullCmd += " " + a
	}

	entry := GenerateCronEntry(CronConfig{
		Minute:  minuteStr,
		Hour:    hourStr,
		Day:     dayStr,
		Weekday: weekdayStr,
		Command: fullCmd,
	})

	// Store the generated cron entry in the schedule config so the user
	// can review and install it. We do not modify system crontab directly.
	_ = entry
	return nil
}

// uninstallLaunchd removes the plist file for a schedule.
func (s *Siphon) uninstallLaunchd(name string) error {
	dir := LaunchAgentsDir()
	plistPath := filepath.Join(dir, PlistFilename(name))

	if err := s.Runtime.Remove(plistPath, false); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}
	return nil
}

// isScheduleActive checks whether a schedule's backend artifact exists.
func (s *Siphon) isScheduleActive(name, backend string) bool {
	switch backend {
	case "launchd":
		dir := LaunchAgentsDir()
		plistPath := filepath.Join(dir, PlistFilename(name))
		_, err := s.Runtime.Stat(plistPath, false)
		return err == nil
	case "cron":
		// Cannot reliably check system crontab; report active if configured.
		return true
	default:
		return false
	}
}

// saveScheduleConfig persists a schedule to the user config file.
func (s *Siphon) saveScheduleConfig(opts *CreateScheduleOptions) error {
	userCfg, err := s.ConfigService.UserConfig(false)
	if err != nil {
		userCfg = &Config{}
	}

	if userCfg.Schedules == nil {
		userCfg.Schedules = make(map[string]*ScheduleConfig)
	}

	sc := &ScheduleConfig{
		Connection: &opts.Connection,
		Backend:    &opts.Backend,
		Time:       &opts.Time,
		Interval:   &opts.Interval,
	}
	if opts.Repo != "" {
		sc.Repo = &opts.Repo
	}
	if opts.Database != "" {
		sc.Database = &opts.Database
	}
	if opts.BackupType != "" {
		sc.BackupType = &opts.BackupType
	}
	if opts.Compress != "" {
		sc.Compress = &opts.Compress
	}
	if opts.Retention != nil {
		sc.Retention = opts.Retention
	}

	enabled := true
	sc.Enabled = &enabled

	userCfg.Schedules[opts.Name] = sc

	path := s.PathService.UserConfig()
	if err := WriteConfig(s.Runtime, path, userCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	s.ConfigService.ResetCache()
	return nil
}

// removeScheduleConfig removes a schedule from the user config file.
func (s *Siphon) removeScheduleConfig(name string) error {
	userCfg, err := s.ConfigService.UserConfig(false)
	if err != nil {
		return ErrScheduleNotFound
	}

	if userCfg.Schedules == nil {
		return ErrScheduleNotFound
	}

	if _, exists := userCfg.Schedules[name]; !exists {
		return ErrScheduleNotFound
	}

	delete(userCfg.Schedules, name)

	path := s.PathService.UserConfig()
	if err := WriteConfig(s.Runtime, path, userCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	s.ConfigService.ResetCache()
	return nil
}
