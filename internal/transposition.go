package internal

type TTFlag int

const (
	TTExact TTFlag = iota
	TTLowerBound
	TTUpperBound
)

type TTEntry struct {
	Hash     uint64
	Depth    int
	Score    int
	Flag     TTFlag
	BestMove *Move
}

type TranspositionTable struct {
	entries []TTEntry
	size    uint64
	mask    uint64
}

func NewTranspositionTable(size uint64) *TranspositionTable {
	actualSize := uint64(1)
	for actualSize*2 <= size {
		actualSize *= 2
	}

	return &TranspositionTable{
		entries: make([]TTEntry, actualSize),
		size:    actualSize,
		mask:    actualSize - 1,
	}
}

func (tt *TranspositionTable) Probe(hash uint64) (TTEntry, bool) {
	idx := hash & tt.mask
	entry := tt.entries[idx]

	if entry.Hash == hash {
		return entry, true
	}
	return TTEntry{}, false
}

func (tt *TranspositionTable) Store(hash uint64, depth, score int, flag TTFlag, bestMove *Move) {
	idx := hash & tt.mask
	existing := tt.entries[idx]

	if existing.Hash == 0 ||
		existing.Hash == hash ||
		depth >= existing.Depth ||
		(existing.Hash != hash && depth >= existing.Depth-2) {
		tt.entries[idx] = TTEntry{
			Hash:     hash,
			Depth:    depth,
			Score:    score,
			Flag:     flag,
			BestMove: bestMove,
		}
	}
}

func (tt *TranspositionTable) Clear() {
	for i := range tt.entries {
		tt.entries[i] = TTEntry{}
	}
}

func (tt *TranspositionTable) GetBestMove(hash uint64) *Move {
	entry, found := tt.Probe(hash)
	if found {
		return entry.BestMove
	}
	return nil
}
