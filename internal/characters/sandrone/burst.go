package sandrone

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/glog"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// The attack starts 0.55s after 0.92 of the burst's animation state: 3 bombardment hits 0.2s
// apart, then the Convective Inhibition Ray 0.85s after the first, whose Radiance state is read
// 0.28s after the first (Avatar_MarionetteNew_ElementalBurst, _BurstAttacker). The animation
// state length (100 frames) and the cancels are estimated.
var burstFrames []int

const (
	burstAttack   = 92 + 33
	burstRadiance = 17
	burstRay      = 51
	burstCD       = 15 * 60
	a1RayPerStack = 0.1
)

var burstBombardHitmarks = []int{0, 12, 24}

func init() {
	burstFrames = frames.InitAbilSlice(100)
	burstFrames[action.ActionAttack] = 92
	burstFrames[action.ActionCharge] = 92
	burstFrames[action.ActionSkill] = 92
	burstFrames[action.ActionDash] = 92
	burstFrames[action.ActionJump] = 92
	burstFrames[action.ActionSwap] = 92
}

func (c *char) Burst(p map[string]int) (action.Info, error) {
	// A1: on cast, Refined Tactics stacks are cleared, each adding 10% to the ray's Stellar
	// Glimmer DMG (Avatar_MarionetteNew_ElementalBurstEnhance: the stacks are spent in any
	// state, the bonus only scales the Stellar-Conduct and Stellar Swirl versions)
	rayMult := 1.0
	if c.Base.Ascension >= 1 {
		stacks := c.refinedTactics()
		rayMult += a1RayPerStack * float64(stacks)
		c.powerLost = 0
		c.powerLostSrc = -1
		c.Core.Log.NewEvent("sandrone consumes refined tactics", glog.LogCharacterEvent, c.Index()).
			Write("stacks", stacks)
	}

	ai := info.AttackInfo{
		ActorIndex: c.Index(),
		AttackTag:  attacks.AttackTagElementalBurst,
		ICDTag:     attacks.ICDTagElementalBurst,
		ICDGroup:   attacks.ICDGroupDefault,
		StrikeType: attacks.StrikeTypeDefault,
		Element:    attributes.Cryo,
		Durability: 25,
		Mult:       bombard[c.TalentLvlBurst()],
	}
	for i, hitmark := range burstBombardHitmarks {
		ai := ai
		ai.Abil = fmt.Sprintf("Bombardment %v", i)
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 6.5}, 7.3)
		c.Core.QueueAttack(ai, ap, burstAttack+hitmark, burstAttack+hitmark)
	}

	c.Core.Tasks.Add(func() {
		ai := ai
		ai.Abil = "Convective Inhibition Ray"
		ai.StrikeType = attacks.StrikeTypeBlunt
		ai.Mult = ray[c.TalentLvlBurst()]
		radiance := c.getRadiance()
		if radiance != radianceNone {
			c.stellarInfo(&ai, ai.Abil, radiance, raySSC, raySSw, c.TalentLvlBurst())
			ai.Mult *= rayMult
		}
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 6.5}, 7.5)
		c.Core.QueueAttack(ai, ap, burstRay-burstRadiance, burstRay-burstRadiance, c.c4CB)
	}, burstAttack+burstRadiance)

	c.SetCD(action.ActionBurst, burstCD)
	c.ConsumeEnergy(5)

	return action.Info{
		Frames:          frames.NewAbilFunc(burstFrames),
		AnimationLength: burstFrames[action.InvalidAction],
		CanQueueAfter:   burstFrames[action.ActionDash],
		State:           action.BurstState,
	}, nil
}
