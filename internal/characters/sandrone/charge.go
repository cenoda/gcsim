package sandrone

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
)

// The charged attack is held: an entry animation (ExtraAttack), then Fagio fires in Decoding
// mode (ExtraAttack1) or Power Overdrive mode (ExtraAttack1Weak) until it is released, then an
// end animation (ExtraAttack3). Sweeping fire starts at 0.65 of the entry (ability data).
// The entry and end lengths are estimated so that a C E pair costs about 1s besides Decoding
// mode: 3[C E] at C0 and 2[C E] at C1 then take 10-13s, the field time the KQM Sandrone quick
// guide reports.
//
// hold: frames Fagio fires after the entry. By default the charged attack is released when Fagio
// enters Power Overdrive (3 condensed beams at C0, 6 at C1, from 0 Decoding Power), or after
// 0.8s (2 Power Overdrive shots) if it is already in Power Overdrive.
var chargeEndFrames []int

const (
	chargeEntry      = 21
	sweepStart       = 14
	maxHold          = 12 * 60 // MarionetteExtraAttackHold
	overdriveHoldDef = 48
)

func init() {
	chargeEndFrames = frames.InitAbilSlice(30)
	chargeEndFrames[action.ActionSkill] = 18
	chargeEndFrames[action.ActionBurst] = 18
	chargeEndFrames[action.ActionDash] = 18
	chargeEndFrames[action.ActionJump] = 18
	chargeEndFrames[action.ActionSwap] = 18
}

func (c *char) ChargeAttack(p map[string]int) (action.Info, error) {
	hold, ok := p["hold"]
	switch {
	case ok:
		hold = min(max(hold, 0), maxHold)
	case c.overdrive:
		hold = overdriveHoldDef
	default:
		hold = c.overdriveAfter()
	}

	src := c.Core.F + 1
	c.caSrc = src
	c.c1OnCharge(src)

	c.Core.Tasks.Add(func() {
		if c.decodingActive(src) {
			c.startSweep(src)
		}
	}, sweepStart)
	c.Core.Tasks.Add(func() {
		if c.caActive(src) {
			c.startDecoding(src)
		}
	}, chargeEntry)
	release := func() {
		if c.caSrc == src {
			c.caSrc = 0
		}
	}
	c.Core.Tasks.Add(release, chargeEntry+hold)

	return action.Info{
		Frames: func(next action.Action) int {
			return chargeEntry + hold + chargeEndFrames[next]
		},
		AnimationLength: chargeEntry + hold + chargeEndFrames[action.InvalidAction],
		CanQueueAfter:   chargeEntry + hold + chargeEndFrames[action.ActionDash],
		State:           action.ChargeAttackState,
		OnRemoved:       func(next action.AnimationState) { release() },
	}, nil
}

// Fagio fires the charged attack, so it costs no stamina (ExtraAttack_Claymore_Stamina_Cost 0)
func (c *char) ActionStam(a action.Action, p map[string]int) float64 {
	if a == action.ActionCharge {
		return 0
	}
	return c.Character.ActionStam(a, p)
}
