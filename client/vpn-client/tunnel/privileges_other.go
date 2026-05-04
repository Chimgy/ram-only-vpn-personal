//go:build !darwin

package tunnel

func EnsurePrivileges() error { return nil }
