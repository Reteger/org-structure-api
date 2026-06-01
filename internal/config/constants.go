package config

const (
	MinNameLength           = 1
	MaxNameLength           = 200
	MinTreeDepth            = 1
	MaxTreeDepth            = 5
	DefaultTreeDepth        = 1
	DefaultIncludeEmployees = true
	PgErrUniqueViolation    = "23505"
	DeleteModeCascade       = "cascade"
	DeleteModeReassign      = "reassign"
)
