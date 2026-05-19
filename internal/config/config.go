package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration.
type Config struct {
	Port                      string
	DatabaseURL               string
	Env                       string
	MagicLinkBaseURL          string
	MagicLinkTTL              time.Duration
	MatchMinOverlapMinutes    int
	PreMatchMessageLimit      int
	MatchRequestMessageMaxLen int
	MatchMessageMaxLen        int
	MatchDailyMsgLimit        int
}

// Load reads configuration from environment variables.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "6176"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "prod"
	}

	magicLinkBaseURL := os.Getenv("MAGIC_LINK_BASE_URL")
	if magicLinkBaseURL == "" {
		magicLinkBaseURL = "http://localhost:3000/auth/verify"
	}

	ttlMinutes := 15
	if raw := os.Getenv("MAGIC_LINK_TTL_MINUTES"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			ttlMinutes = value
		}
	}

	magicLinkTTL := time.Duration(ttlMinutes) * time.Minute

	minOverlap := 60
	if raw := os.Getenv("MATCH_MIN_OVERLAP_MINUTES"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			minOverlap = value
		}
	}

	preMatchMsgLimit := 5
	if raw := os.Getenv("PRE_MATCH_MESSAGE_LIMIT"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			preMatchMsgLimit = value
		}
	}

	matchReqMsgMaxLen := 500
	if raw := os.Getenv("MATCH_REQUEST_MESSAGE_MAX_LENGTH"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			matchReqMsgMaxLen = value
		}
	}

	matchMsgMaxLen := 2000
	if raw := os.Getenv("MATCH_MESSAGE_MAX_LENGTH"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			matchMsgMaxLen = value
		}
	}

	matchDailyMsgLimit := 200
	if raw := os.Getenv("MATCH_DAILY_MESSAGE_LIMIT"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			matchDailyMsgLimit = value
		}
	}

	return Config{
		Port:                      port,
		DatabaseURL:               os.Getenv("DATABASE_URL"),
		Env:                       env,
		MagicLinkBaseURL:          magicLinkBaseURL,
		MagicLinkTTL:              magicLinkTTL,
		MatchMinOverlapMinutes:    minOverlap,
		PreMatchMessageLimit:      preMatchMsgLimit,
		MatchRequestMessageMaxLen: matchReqMsgMaxLen,
		MatchMessageMaxLen:        matchMsgMaxLen,
		MatchDailyMsgLimit:        matchDailyMsgLimit,
	}
}
