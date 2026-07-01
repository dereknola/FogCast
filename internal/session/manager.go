package session

import "sync"

const serverVersion = "dev"
const (
	defaultMaskSize = 512
	minMaskSize     = 128
	maxMaskSize     = 2048
)

type Manager struct {
	mu    sync.RWMutex
	state State
	mask  []byte
}

type State struct {
	ActiveMap     *MapState       `json:"activeMap"`
	PlayerView    PlayerViewState `json:"playerView"`
	Mask          MaskState       `json:"mask"`
	ServerVersion string          `json:"serverVersion"`
}

type PlayerViewState struct {
	Scale   float64 `json:"scale"`
	OffsetX int     `json:"offsetX"`
	OffsetY int     `json:"offsetY"`
}

type MaskPatch struct {
	X      int
	Y      int
	Width  int
	Height int
	Data   []byte
}

type MapState struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type MaskState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func NewManager(initialMap *MapState) *Manager {
	return NewManagerWithMaskSize(initialMap, defaultMaskSize)
}

func NewManagerWithMaskSize(initialMap *MapState, requestedMaskSize int) *Manager {
	maskSize := normalizeMaskSize(requestedMaskSize)
	mask := make([]byte, maskSize*maskSize)

	return &Manager{
		state: State{
			ActiveMap: cloneMapState(initialMap),
			PlayerView: PlayerViewState{
				Scale:   1,
				OffsetX: 0,
				OffsetY: 0,
			},
			Mask: MaskState{
				Width:  maskSize,
				Height: maskSize,
			},
			ServerVersion: serverVersion,
		},
		mask: mask,
	}
}

func normalizeMaskSize(value int) int {
	if value < minMaskSize {
		return defaultMaskSize
	}
	if value > maxMaskSize {
		return maxMaskSize
	}
	return value
}

func (m *Manager) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.state
}

func (m *Manager) SetActiveMap(activeMap *MapState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.ActiveMap = cloneMapState(activeMap)
}

func (m *Manager) SetPlayerView(view PlayerViewState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.PlayerView = view
}

func (m *Manager) MaskCopy() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	copyMask := make([]byte, len(m.mask))
	copy(copyMask, m.mask)
	return copyMask
}

func (m *Manager) SetMask(mask []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(mask) != len(m.mask) {
		return false
	}

	copy(m.mask, mask)
	return true
}

func (m *Manager) ApplyMaskPatch(patch MaskPatch) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if patch.Width <= 0 || patch.Height <= 0 || patch.X < 0 || patch.Y < 0 {
		return false
	}

	maskWidth := m.state.Mask.Width
	maskHeight := m.state.Mask.Height
	if patch.X+patch.Width > maskWidth || patch.Y+patch.Height > maskHeight {
		return false
	}

	expected := patch.Width * patch.Height
	if len(patch.Data) != expected {
		return false
	}

	for row := 0; row < patch.Height; row += 1 {
		srcStart := row * patch.Width
		dstStart := (patch.Y+row)*maskWidth + patch.X
		copy(m.mask[dstStart:dstStart+patch.Width], patch.Data[srcStart:srcStart+patch.Width])
	}

	return true
}

func (m *Manager) MaskLength() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.mask)
}

func (m *Manager) RevealAll() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.mask {
		m.mask[i] = 255
	}

	copyMask := make([]byte, len(m.mask))
	copy(copyMask, m.mask)
	return copyMask
}

func (m *Manager) ShroudAll() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.mask {
		m.mask[i] = 0
	}

	copyMask := make([]byte, len(m.mask))
	copy(copyMask, m.mask)
	return copyMask
}

func cloneMapState(input *MapState) *MapState {
	if input == nil {
		return nil
	}

	copy := *input
	return &copy
}
