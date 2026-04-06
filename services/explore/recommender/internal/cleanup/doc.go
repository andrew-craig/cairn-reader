// Package cleanup provides background jobs for article retention management.
// It implements two-phase deletion: soft-deleting articles after a configurable
// retention period, then hard-deleting them after an additional grace period.
package cleanup
