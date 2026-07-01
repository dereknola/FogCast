package session

import "sync"

const serverVersion = "dev"

type Manager struct {
	mu    sync.RWMutex
	state State
	mask  []byte
}

type State struct {
	ActiveMap     *MapState `json:"activeMap"`
	Mask          MaskState `json:"mask"`
	ServerVersion string    `json:"serverVersion"`
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
	mask := make([]byte, 512*512)

	return &Manager{
		state: State{
			ActiveMap: cloneMapState(initialMap),
			Mask: MaskState{
				Width:  512,
				Height: 512,
			},
			ServerVersion: serverVersion,
		},
		mask: mask,
	}
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
