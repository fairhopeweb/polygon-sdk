package ibft

import (
	"fmt"

	"github.com/0xPolygon/polygon-sdk/types"
)

// PoAMechanism defines specific hooks for the Proof of Authority IBFT mechanism
type PoAMechanism struct {
	// Reference to the main IBFT implementation
	ibft *Ibft

	// hookMap is the collection of registered hooks
	hookMap map[string]func(interface{}) error

	// Used for easy lookups
	mechanismType Type
}

// PoAFactory initializes the required data
// for the Proof of Authority mechanism
func PoAFactory() (ConsensusMechanism, error) {
	poa := &PoAMechanism{
		mechanismType: PoA,
	}

	// Initialize the hook map
	poa.initializeHookMap()

	return poa, nil
}

// acceptStateLogHook logs the current snapshot with the number of votes
func (poa *PoAMechanism) acceptStateLogHook(snapParam interface{}) error {
	// Cast the param to a *Snapshot
	snap := snapParam.(*Snapshot)

	// Log the info message
	poa.ibft.logger.Info(
		"current snapshot",
		"validators",
		len(snap.Set),
		"votes",
		len(snap.Votes),
	)

	return nil
}

// verifyHeadersHook verifies that the header nonce conforms to the IBFT PoA proposal format
func (poa *PoAMechanism) verifyHeadersHook(nonceParam interface{}) error {
	// Cast the param to the nonce
	nonce := nonceParam.(types.Nonce)

	// Check the nonce format
	// Because you must specify either AUTH or DROP vote, it is confusing how to have a block without any votes.
	// 		This is achieved by specifying the miner field to zeroes,
	// 		because then the value in the Nonce will not be taken into consideration.
	if nonce != nonceDropVote && nonce != nonceAuthVote {
		return fmt.Errorf("invalid nonce")
	}

	return nil
}

// processHeadersHookParams are the params passed into the processHeadersHook
type processHeadersHookParams struct {
	header     *types.Header
	snap       *Snapshot
	parentSnap *Snapshot
	proposer   types.Address
	saveSnap   func(h *types.Header)
}

// processHeadersHook does the required logic for PoA header processing
func (poa *PoAMechanism) processHeadersHook(hookParam interface{}) error {
	// Cast the params to processHeadersHookParams
	params := hookParam.(*processHeadersHookParams)
	number := params.header.Number

	if number%poa.ibft.epochSize == 0 {
		// during a checkpoint block, we reset the votes
		// and there cannot be any proposals
		params.snap.Votes = nil
		params.saveSnap(params.header)

		// remove in-memory snapshots from two epochs before this one
		epoch := int(number/poa.ibft.epochSize) - 2
		if epoch > 0 {
			purgeBlock := uint64(epoch) * poa.ibft.epochSize
			poa.ibft.store.deleteLower(purgeBlock)
		}
		return nil
	}

	// if we have a miner address, this might be a vote
	if params.header.Miner == types.ZeroAddress {
		return nil
	}

	// the nonce selects the action
	var authorize bool
	switch {
	case params.header.Nonce == nonceAuthVote:
		authorize = true
	case params.header.Nonce == nonceDropVote:
		authorize = false
	default:
		return fmt.Errorf("incorrect vote nonce")
	}

	// validate the vote
	if authorize {
		// we can only authorize if they are not on the validators list
		if params.snap.Set.Includes(params.header.Miner) {
			return nil
		}
	} else {
		// we can only remove if they are part of the validators list
		if !params.snap.Set.Includes(params.header.Miner) {
			return nil
		}
	}

	voteCount := params.snap.Count(func(v *Vote) bool {
		return v.Validator == params.proposer && v.Address == params.header.Miner
	})

	if voteCount > 1 {
		// there can only be one vote per validator per address
		return fmt.Errorf("more than one proposal per validator per address found")
	}
	if voteCount == 0 {
		// cast the new vote since there is no one yet
		params.snap.Votes = append(params.snap.Votes, &Vote{
			Validator: params.proposer,
			Address:   params.header.Miner,
			Authorize: authorize,
		})
	}

	// check the tally for the proposed validator
	tally := params.snap.Count(func(v *Vote) bool {
		return v.Address == params.header.Miner
	})

	// If more than a half of all validators voted
	if tally > params.snap.Set.Len()/2 {
		if authorize {
			// add the candidate to the validators list
			params.snap.Set.Add(params.header.Miner)
		} else {
			// remove the candidate from the validators list
			params.snap.Set.Del(params.header.Miner)

			// remove any votes casted by the removed validator
			params.snap.RemoveVotes(func(v *Vote) bool {
				return v.Validator == params.header.Miner
			})
		}

		// remove all the votes that promoted this validator
		params.snap.RemoveVotes(func(v *Vote) bool {
			return v.Address == params.header.Miner
		})
	}

	return nil
}

// initializeHookMap registers the hooks that the PoA mechanism
// should have
func (poa *PoAMechanism) initializeHookMap() {
	// Create the hook map
	poa.hookMap = make(map[string]func(interface{}) error)

	// Register the AcceptStateLogHook
	poa.hookMap[AcceptStateLogHook] = poa.acceptStateLogHook

	// Register the VerifyHeadersHook
	poa.hookMap[VerifyHeadersHook] = poa.verifyHeadersHook

	// Register the ProcessHeadersHook
	poa.hookMap[ProcessHeadersHook] = poa.processHeadersHook
}

// GetType implements the ConsensusMechanism interface method
func (poa *PoAMechanism) GetType() Type {
	return poa.mechanismType
}

// GetHookMap implements the ConsensusMechanism interface method
func (poa *PoAMechanism) GetHookMap() map[string]func(interface{}) error {
	return poa.hookMap
}
