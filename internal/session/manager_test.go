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

func TestApplyMaskPatchUpdatesEntireRectangle(t *testing.T) {
	manager := NewManagerWithMaskSize(nil, 32)

	patch := MaskPatch{
		X:      5,
		Y:      7,
		Width:  7,
		Height: 5,
		Data:   make([]byte, 7*5),
	}
	for i := range patch.Data {
		patch.Data[i] = 211
	}

	if ok := manager.ApplyMaskPatch(patch); !ok {
		t.Fatalf("expected patch application to succeed")
	}

	mask := manager.MaskCopy()
	stride := manager.State().Mask.Width
	for y := 0; y < manager.State().Mask.Height; y += 1 {
		for x := 0; x < manager.State().Mask.Width; x += 1 {
			value := mask[y*stride+x]
			inside := x >= patch.X && x < patch.X+patch.Width && y >= patch.Y && y < patch.Y+patch.Height
			if inside && value != 211 {
				t.Fatalf("expected patched cell (%d,%d) to be 211, got %d", x, y, value)
			}
			if !inside && value != 0 {
				t.Fatalf("expected non-patched cell (%d,%d) to remain 0, got %d", x, y, value)
			}
		}
	}
}
