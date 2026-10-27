package vodyanitsa

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// The burst hits once (5 m around her, 50 Hydro gauge, no ICD tag) when
// Avatar_Vodyanitsa_ElementalBurst_Attack ends, at 95% of the burst state; whether Song of Ages
// Past is active is read then, and multiplies the talent multiplier by 1 + burstBonus. The burst
// state's length is not in the data: it is estimated at 100 frames, so the hit lands at 95, and the
// animation at 110 frames, close to other catalyst bursts.
var burstFrames []int

const (
	burstHitmark = 95
	burstCD      = 15 * 60
)

func init() {
	burstFrames = frames.InitAbilSlice(110)
	burstFrames[action.ActionSwap] = 100
	burstFrames[action.ActionDash] = 100
	burstFrames[action.ActionJump] = 100
}

func (c *char) Burst(p map[string]int) (action.Info, error) {
	c.Core.Tasks.Add(func() {
		mult := burst[c.TalentLvlBurst()]
		abil := "Koda: Sink With Thee"
		if c.StatusIsActive(songKey) {
			mult *= 1 + burstBonus[c.TalentLvlBurst()]
			abil += " (Song of Ages Past)"
		}
		ai := info.AttackInfo{
			ActorIndex: c.Index(),
			Abil:       abil,
			AttackTag:  attacks.AttackTagElementalBurst,
			ICDTag:     attacks.ICDTagNone,
			ICDGroup:   attacks.ICDGroupDefault,
			StrikeType: attacks.StrikeTypeDefault,
			Element:    attributes.Hydro,
			Durability: 50,
			Mult:       mult,
			UseHP:      true,
		}
		c.Core.QueueAttack(ai, combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, 5), 0, 0)
	}, burstHitmark)

	c.SetCD(action.ActionBurst, burstCD)
	c.ConsumeEnergy(5)

	return action.Info{
		Frames:          frames.NewAbilFunc(burstFrames),
		AnimationLength: burstFrames[action.InvalidAction],
		CanQueueAfter:   burstFrames[action.ActionSwap],
		State:           action.BurstState,
	}, nil
}
