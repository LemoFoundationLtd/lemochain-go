package account

import (
	"github.com/LemoFoundationLtd/lemochain-go/chain/types"
	"github.com/LemoFoundationLtd/lemochain-go/common"
	"github.com/LemoFoundationLtd/lemochain-go/common/log"
	"sort"
)

type revision struct {
	id           int
	journalIndex int
}

type RawAccountLoader interface {
	getRawAccount(address common.Address) *Account
}

// LogProcessor records change logs and contract events during block's transaction execution. It can access raw account data for undo/redo
type LogProcessor struct {
	accountLoader RawAccountLoader

	// The change logs generated by transactions from the same block. We put all logs in a queue so we can revert to any revision
	changeLogs     []*types.ChangeLog
	revisions      []revision
	nextRevisionId int

	// This map holds contract events generated by every transactions
	events []*types.Event
}

// NewLogProcessor creates a new LogProcessor. It is used to maintain change logs queue
func NewLogProcessor(accountLoader RawAccountLoader) *LogProcessor {
	if accountLoader == nil {
		panic("account.NewLogProcessor is called without an RawAccountLoader")
	}
	return &LogProcessor{
		accountLoader: accountLoader,
		changeLogs:    make([]*types.ChangeLog, 0),
		events:        make([]*types.Event, 0),
	}
}

func (h *LogProcessor) GetAccount(addr common.Address) types.AccountAccessor {
	return h.accountLoader.getRawAccount(addr)
}

func (h *LogProcessor) PushEvent(event *types.Event) {
	h.events = append(h.events, event)
}

func (h *LogProcessor) PopEvent() error {
	size := len(h.events)
	if size == 0 {
		return ErrNoEvents
	}
	h.events = h.events[:size-1]
	return nil
}

func (h *LogProcessor) GetEvents() []*types.Event {
	return h.events
}

func (h *LogProcessor) PushChangeLog(log *types.ChangeLog) {
	h.changeLogs = append(h.changeLogs, log)
}

func (h *LogProcessor) GetChangeLogs() []*types.ChangeLog {
	return h.changeLogs
}

func (h *LogProcessor) GetLogsByAddress(address common.Address) []*types.ChangeLog {
	result := make([]*types.ChangeLog, 0)
	for _, changeLog := range h.changeLogs {
		if changeLog.Address == address {
			result = append(result, changeLog)
		}
	}
	return result
}

// GetNextVersion generate new version for change log
func (h *LogProcessor) GetNextVersion(logType types.ChangeLogType, addr common.Address) uint32 {
	// read current version in account
	maxVersion := h.GetAccount(addr).GetBaseVersion(logType)
	// find newest change log in LogProcessor
	for _, changeLog := range h.changeLogs {
		if changeLog.LogType == logType && changeLog.Address == addr {
			maxVersion = changeLog.Version
		}
	}
	return maxVersion + 1
}

func (h *LogProcessor) Clear() {
	h.changeLogs = make([]*types.ChangeLog, 0)
	h.events = make([]*types.Event, 0)
	h.nextRevisionId = 0
	h.revisions = make([]revision, 0)
}

// Snapshot returns an identifier for the current revision of the change log.
func (h *LogProcessor) Snapshot() int {
	id := h.nextRevisionId
	h.nextRevisionId++
	h.revisions = append(h.revisions, revision{id, len(h.changeLogs)})
	return id
}

// checkRevisionAvailable check if the newest revision is accessible
func (h *LogProcessor) checkRevisionAvailable() bool {
	if len(h.revisions) == 0 {
		return true
	}
	last := h.revisions[len(h.revisions)-1]
	return last.journalIndex <= len(h.changeLogs)
}

// RevertToSnapshot reverts all changes made since the given revision.
func (h *LogProcessor) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(h.revisions), func(i int) bool {
		return h.revisions[i].id >= revid
	})
	if idx == len(h.revisions) || h.revisions[idx].id != revid {
		log.Errorf("revision id %v cannot be reverted", revid)
		panic(ErrRevisionNotExist)
	}
	snapshot := h.revisions[idx].journalIndex

	lastVersions := make(map[common.Address]uint32)
	// Replay the change log to undo changes.
	for i := len(h.changeLogs) - 1; i >= snapshot; i-- {
		changeLog := h.changeLogs[i]
		account := h.GetAccount(changeLog.Address)

		// Make sure the sequence of changelog is correct. And the change log's version is bigger than in account, because the version in account is only updated at the end of block process
		accountVersion := account.(*Account).GetBaseVersion(changeLog.LogType)
		last, ok := lastVersions[changeLog.Address]
		// Check change log's version
		if (!ok || last-1 == changeLog.Version) && accountVersion < changeLog.Version {
			lastVersions[changeLog.Address] = changeLog.Version
		} else {
			log.Errorf("expected undo version %d, got %d", last-1, changeLog.Version)
			panic(types.ErrWrongChangeLogVersion)
		}

		err := changeLog.Undo(h)
		if err != nil {
			panic(err)
		}
	}
	h.changeLogs = h.changeLogs[:snapshot]

	// Remove invalidated snapshots from the stack.
	h.revisions = h.revisions[:idx]
}

// Rebuild loads and redo all change logs to update account to the newest state.
func (h *LogProcessor) Rebuild(address common.Address, logs types.ChangeLogSlice) (*Account, error) {
	sort.Sort(logs)
	account := h.GetAccount(address).(*Account)

	lastVersions := make(map[common.Address]uint32)
	// Replay the change log to undo changes.
	for _, changeLog := range logs {
		accountVersion := account.GetBaseVersion(changeLog.LogType)
		last, ok := lastVersions[changeLog.Address]
		// Make sure the sequence of changelog is correct. And the change log's version is bigger than in account
		if !ok {
			lastVersions[changeLog.Address] = accountVersion
			last = accountVersion
		}
		// Check change log's version
		if changeLog.Version < last+1 {
			log.Errorf("expected redo version %d, got %d", last+1, changeLog.Version)
			panic(types.ErrAlreadyRedo)
		} else if changeLog.Version > last+1 {
			log.Errorf("expected redo version %d, got %d", last+1, changeLog.Version)
			panic(types.ErrWrongChangeLogVersion)
		}
		last = changeLog.Version

		err := changeLog.Redo(h)
		if err != nil && err != types.ErrAlreadyRedo {
			panic(err)
		}
	}

	// save account
	return account, nil
}

// MergeChangeLogs merges the change logs for same account in block
func (h *LogProcessor) MergeChangeLogs(fromIndex int) {
	needMerge := h.changeLogs[fromIndex:]
	mergedLogs := MergeChangeLogs(needMerge)
	h.changeLogs = append(h.changeLogs[:fromIndex], mergedLogs...)
	// make sure the snapshot still work
	if !h.checkRevisionAvailable() {
		log.Error("invalid revision", "last revision", h.revisions[len(h.revisions)-1], "new logs count", len(h.changeLogs), "from index", fromIndex, "merged logs count", len(mergedLogs))
		panic(ErrSnapshotIsBroken)
	}
}