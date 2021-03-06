package store

import (
	"bytes"
	"github.com/LemoFoundationLtd/lemochain-core/chain/types"
	"github.com/LemoFoundationLtd/lemochain-core/common"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"math/big"
)

type CBlock struct {
	Block           *types.Block
	AccountTrieDB   *AccountTrieDB
	CandidateTrieDB *CandidateTrieDB
	Top             *VoteTop
	Parent          *CBlock
	Children        []*CBlock
}

func NewGenesisBlock(block *types.Block, beansdb *BeansDB) *CBlock {
	return &CBlock{
		Block:           block,
		AccountTrieDB:   NewEmptyAccountTrieDB(beansdb),
		CandidateTrieDB: NewEmptyCandidateTrieDB(),
		Top:             NewEmptyVoteTop(),
	}
}

func NewNormalBlock(block *types.Block, accountTrieDB *AccountTrieDB, candidateTrieDB *CandidateTrieDB, top *VoteTop) *CBlock {
	return &CBlock{
		Block:           block,
		AccountTrieDB:   accountTrieDB.Clone(),
		CandidateTrieDB: candidateTrieDB.Clone(),
		Top:             top.Clone(),
	}
}

func (block *CBlock) existCandidateProfile(account *types.AccountData) bool {
	if (account == nil) ||
		(len(account.Candidate.Profile) <= 0) {
		return false
	}
	return true
}

func (block *CBlock) filterCandidates(accounts []*types.AccountData) []*Candidate {
	candidates := make([]*Candidate, 0)
	for index := 0; index < len(accounts); index++ {
		account := accounts[index]
		if block.existCandidateProfile(account) {
			candidates = append(candidates, &Candidate{
				Address: account.Address,
				Total:   new(big.Int).Set(account.Candidate.Votes),
			})
		}
	}
	return candidates
}

func (block *CBlock) dye(candidates []*Candidate) {
	if len(candidates) <= 0 {
		return
	}

	for index := 0; index < len(candidates); index++ {
		candidate := candidates[index]
		block.CandidateTrieDB.Put(candidate, block.Block.Height())
	}
}

func (block *CBlock) lessThan30(oldCandidates []*Candidate, newCandidates []*Candidate) {
	block.Top.Rank(max_candidate_count, append(oldCandidates, newCandidates...))
}

func (block *CBlock) minIsIncrease(oldMin *Candidate, newMin *Candidate) bool {
	if (oldMin != nil) && (newMin != nil) && (newMin.Total.Cmp(oldMin.Total) >= 0) {
		return true
	} else {
		return false
	}
}

func (block *CBlock) canPick(src *Candidate, dst *Candidate) bool {
	if (src.Total.Cmp(dst.Total) < 0) ||
		((src.Total.Cmp(dst.Total) == 0) && (bytes.Compare(src.Address[:], dst.Address[:]) < 0)) {
		return true
	} else {
		return false
	}
}

func (block *CBlock) updateTop(changedCandidates []*Candidate) {
	newTop := block.Top.Clone()
	// remove unregistered candidates
	unregisters := block.collectUnregisters()
	newTop.Top = filterUnregisters(newTop.Top, unregisters)
	changedCandidates = filterUnregisters(changedCandidates, unregisters)

	newTop.MergeCandidates(changedCandidates)

	if block.Top.Count() < max_candidate_count {
		block.Top = newTop
	} else if block.Top.Count() > newTop.Count() {
		// some candidates unregistered. so maybe some normal nodes will become new candidates
		// resort all candidates
		block.Top.Rank(max_candidate_count, block.CandidateTrieDB.GetAll())
	} else if newTop.Min().Total.Cmp(block.Top.Min().Total) >= 0 {
		// the min votes become bigger, it means some old candidates get richer now.
		// the other candidates whose vote is not changed, must not be in the top list. so we can just use the newTop
		block.Top = newTop
	} else {
		// the min votes become smaller, it means some old candidates lose their vote.
		// maybe the loser candidates will become normal nodes, and some normal nodes will become new candidates
		// resort all candidates
		block.Top.Rank(max_candidate_count, block.CandidateTrieDB.GetAll())
	}
}

func (block *CBlock) Ranking(voteLogs types.ChangeLogSlice) {
	if len(voteLogs) <= 0 {
		return
	}
	// collect changed candidates
	changedCandidates := make([]*Candidate, 0)
	for _, changelog := range voteLogs {
		newVote, ok := changelog.NewVal.(big.Int)
		if !ok {
			log.Error("vote log is required!", "changLog", changelog.String())
			continue
		}
		changedCandidates = append(changedCandidates, &Candidate{
			Address: changelog.Address,
			Total:   new(big.Int).Set(&newVote),
		})
	}
	// update candidates' data in global list
	block.dye(changedCandidates)
	// update top 30
	block.updateTop(changedCandidates)
}

func filterUnregisters(candidates []*Candidate, unregisters map[common.Address]bool) []*Candidate {
	if len(unregisters) <= 0 {
		return candidates
	}

	newCandidates := make([]*Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := unregisters[candidate.Address]; !ok {
			newCandidates = append(newCandidates, candidate)
		}
	}
	return newCandidates
}

func (block *CBlock) collectUnregisters() map[common.Address]bool {
	height := block.Block.Height()
	accounts := block.AccountTrieDB.Collect(height)
	unregisterMap := make(map[common.Address]bool, 0)
	for _, account := range accounts {
		result, ok := account.Candidate.Profile[types.CandidateKeyIsCandidate]
		if ok && result == types.NotCandidateNode {
			unregisterMap[account.Address] = true
		}
	}

	return unregisterMap
}

func (block *CBlock) BeChildOf(parent *CBlock) {
	block.Parent = parent

	if parent != nil {
		// check if exist
		for _, child := range parent.Children {
			if child == block {
				return
			}
		}

		parent.Children = append(parent.Children, block)
	}
}

func (block *CBlock) IsSameBlock(b *CBlock) bool {
	if block == b {
		return true
	}
	if block == nil || b == nil {
		return false
	}
	return block.Block.Hash() == b.Block.Hash()
}

// CollectToParent collect blocks from parent to parent, include itself and exclude the end block
func (block *CBlock) CollectToParent(end *CBlock) []*CBlock {
	blocks := make([]*CBlock, 0)
	for iter := block; iter != end && iter != nil; iter = iter.Parent {
		blocks = append(blocks, iter)
	}
	return blocks
}

// Walk iterate every child recursively. Not include itself
func (block *CBlock) Walk(fn func(*CBlock), exclude *CBlock) {
	for _, child := range block.Children {
		if exclude == nil || !child.IsSameBlock(exclude) {
			fn(child)
			child.Walk(fn, exclude)
		}
	}
}
