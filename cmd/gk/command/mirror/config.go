package mirror

type Config struct {
	Z1, Z2         string
	C1, C2         string
	ExcludedTopics map[string]struct{}
	TopicsOnly     map[string]struct{}
	Compress       string
	BandwidthLimit int64
	Debug          bool
	AutoCommit     bool
	ProgressStep   int64
	ShowStatus     bool
}

func DefaultConfig() *Config {
	return &Config{
		Debug:          false,
		ShowStatus:     false,
		AutoCommit:     true,
		ProgressStep:   5000,
		ExcludedTopics: make(map[string]struct{}),
		TopicsOnly:     make(map[string]struct{}),
	}
}

func (c *Config) realTopics(topics []string) []string {
	r := make([]string, 0, len(topics))
	if len(c.TopicsOnly) > 0 {
		// higher priority over exclusion
		for _, t := range topics {
			if _, present := c.TopicsOnly[t]; present {
				if _, internal := internalTopics[t]; !internal {
					r = append(r, t)
				}
			}
		}

		return r
	} else if len(c.ExcludedTopics) > 0 {
		for _, t := range topics {
			if _, present := c.ExcludedTopics[t]; !present {
				if _, internal := internalTopics[t]; !internal {
					r = append(r, t)
				}
			}
		}

		return r
	} else {
		for _, t := range topics {
			if _, internal := internalTopics[t]; !internal {
				r = append(r, t)
			}
		}
	}

	return r
}
