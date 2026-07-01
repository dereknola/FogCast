package session

import "testing"

func TestNewManagerWithMaskSizeUsesConfiguredSize(t *testing.T) {
	manager := NewManagerWithMaskSize(nil, 768)
	state := manager.State()

	if state.Mask.Width != 768 || state.Mask.Height != 768 {
		t.Fatalf("expected 768x768 mask, got %dx%d", state.Mask.Width, state.Mask.Height)
	}
	if manager.MaskLength() != 768*768 {
		t.Fatalf("expected mask length %d, got %d", 768*768, manager.MaskLength())
	}
}

func TestNewManagerWithMaskSizeClampsOutOfRange(t *testing.T) {
	t.Run("too small falls back to default", func(t *testing.T) {
		manager := NewManagerWithMaskSize(nil, 32)
		state := manager.State()
		if state.Mask.Width != defaultMaskSize || state.Mask.Height != defaultMaskSize {
			t.Fatalf("expected default mask size %d, got %dx%d", defaultMaskSize, state.Mask.Width, state.Mask.Height)
		}
	})

	t.Run("too large clamps to max", func(t *testing.T) {
		manager := NewManagerWithMaskSize(nil, 9999)
		state := manager.State()
		if state.Mask.Width != maxMaskSize || state.Mask.Height != maxMaskSize {
			t.Fatalf("expected max mask size %d, got %dx%d", maxMaskSize, state.Mask.Width, state.Mask.Height)
		}
	})
}
