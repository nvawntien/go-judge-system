package entity

import "testing"

func TestUserSuspensionIsSeparateFromActivation(t *testing.T) {
	user := &User{}
	user.Suspend()
	if !user.IsSuspended || user.IsActive {
		t.Fatalf("after Suspend() = active:%t suspended:%t, want false/true", user.IsActive, user.IsSuspended)
	}

	user.Activate()
	if !user.IsSuspended || !user.IsActive {
		t.Fatalf("after Activate() = active:%t suspended:%t, want true/true", user.IsActive, user.IsSuspended)
	}

	user.Unsuspend()
	if !user.IsActive || user.IsSuspended {
		t.Fatalf("after Unsuspend() = active:%t suspended:%t, want true/false", user.IsActive, user.IsSuspended)
	}
}
