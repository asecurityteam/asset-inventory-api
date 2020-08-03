package logs

// MigrationSuccess is logged for successful runs of schema changes
type MigrationSuccess struct {
	Message string `logevent:"message,default=migration-success"`
	Reason  string `logevent:"reason"`
}
